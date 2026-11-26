/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package gitops performs git clone/fetch/checkout/commit/push in-process via
// go-git, with SSH authentication built entirely in memory from a Flux-style
// deploy-key Secret (identity + known_hosts). It exists so the controller image
// can stay distroless: no `git`/`ssh` binary, no ephemeral private key written
// to disk (which readOnlyRootFilesystem would reject). Both the Deployment
// deploy phase (pkg/apis/deployment) and the manifests-repo bootstrap
// (environments-controller) use it.
package gitops

import (
	"bytes"
	"fmt"
	"net"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
)

// Auth builds an in-memory SSH AuthMethod from a PEM private key and a
// known_hosts blob. Nothing touches disk. A nil/empty identity yields a nil
// AuthMethod, which go-git accepts for anonymous (e.g. local file://) remotes.
func Auth(identity, knownHosts []byte) (transport.AuthMethod, error) {
	if len(bytes.TrimSpace(identity)) == 0 {
		return nil, nil
	}
	auth, err := gitssh.NewPublicKeys("git", identity, "")
	if err != nil {
		return nil, fmt.Errorf("parse ssh private key: %w", err)
	}
	cb, err := hostKeyCallback(knownHosts)
	if err != nil {
		return nil, err
	}
	auth.HostKeyCallback = cb
	return auth, nil
}

// hostKeyCallback parses known_hosts in memory and returns a callback that
// accepts a host key only when it matches one of the parsed entries.
func hostKeyCallback(knownHosts []byte) (ssh.HostKeyCallback, error) {
	var keys []ssh.PublicKey
	rest := knownHosts
	for len(bytes.TrimSpace(rest)) > 0 {
		_, _, pub, _, next, err := ssh.ParseKnownHosts(rest)
		if err != nil {
			break
		}
		if pub != nil {
			keys = append(keys, pub)
		}
		rest = next
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no usable entries in known_hosts")
	}
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		marshalled := key.Marshal()
		for _, k := range keys {
			if bytes.Equal(k.Marshal(), marshalled) {
				return nil
			}
		}
		return fmt.Errorf("host key verification failed: unknown key")
	}, nil
}

// CloneOrOpen opens repoPath when it already holds a clone, otherwise clones
// url into it. An empty remote clones successfully with no HEAD, matching the
// bootstrap's "clone, then create the first branch" flow.
func CloneOrOpen(repoPath, url string, auth transport.AuthMethod) (*gogit.Repository, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err == nil {
		return repo, nil
	}
	if err != gogit.ErrRepositoryNotExists {
		return nil, fmt.Errorf("open %s: %w", repoPath, err)
	}
	repo, err = gogit.PlainClone(repoPath, false, &gogit.CloneOptions{URL: url, Auth: auth})
	if err != nil && err != transport.ErrEmptyRemoteRepository {
		return nil, fmt.Errorf("clone %s: %w", url, err)
	}
	if repo == nil {
		// Empty remote: initialise a local repo with the same origin so a
		// first commit + push can create the branch.
		repo, err = gogit.PlainInit(repoPath, false)
		if err != nil {
			return nil, fmt.Errorf("init %s: %w", repoPath, err)
		}
		if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}}); err != nil {
			return nil, fmt.Errorf("add origin: %w", err)
		}
	}
	return repo, nil
}

// FetchCheckout fetches origin and checks out ref (a branch, tag or commit
// SHA) as a detached HEAD, mirroring the old `fetch origin <ref>` +
// `checkout FETCH_HEAD`. A ref that cannot be resolved after fetch is an error.
func FetchCheckout(repo *gogit.Repository, ref string, auth transport.AuthMethod) error {
	err := repo.Fetch(&gogit.FetchOptions{
		Auth:     auth,
		RefSpecs: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
		Tags:     gogit.AllTags,
		Force:    true,
	})
	if err != nil && err != gogit.NoErrAlreadyUpToDate && err != transport.ErrEmptyRemoteRepository {
		return fmt.Errorf("fetch: %w", err)
	}

	hash, rerr := resolve(repo, ref)
	if rerr != nil {
		return rerr
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	if err := wt.Checkout(&gogit.CheckoutOptions{Hash: hash, Force: true}); err != nil {
		return fmt.Errorf("checkout %s: %w", ref, err)
	}
	return nil
}

// resolve turns ref into a commit hash, trying it verbatim (SHA, tag, local
// ref) and then as a remote-tracking branch (origin/<ref>).
func resolve(repo *gogit.Repository, ref string) (plumbing.Hash, error) {
	for _, rev := range []plumbing.Revision{plumbing.Revision(ref), plumbing.Revision("refs/remotes/origin/" + ref), plumbing.Revision("origin/" + ref)} {
		if h, err := repo.ResolveRevision(rev); err == nil && h != nil {
			return *h, nil
		}
	}
	return plumbing.ZeroHash, fmt.Errorf("resolve ref %q", ref)
}

// StageCommitPush stages addPath (relative to the worktree root), and when the
// index differs from HEAD commits with the given identity and pushes the new
// commit to refs/heads/branch on origin. It returns committed=false (and does
// not push) when the worktree is already clean, matching the old
// `git diff --cached --quiet` short-circuit. The push uses an explicit
// HEAD:refs/heads/<branch> refspec so a detached HEAD still lands on the branch.
func StageCommitPush(repo *gogit.Repository, addPath, message, branch string, auth transport.AuthMethod, name, email string) (bool, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("worktree: %w", err)
	}
	if err := wt.AddWithOptions(&gogit.AddOptions{Path: addPath, All: true}); err != nil {
		return false, fmt.Errorf("stage %s: %w", addPath, err)
	}
	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("status: %w", err)
	}
	if status.IsClean() {
		return false, nil
	}
	sig := &object.Signature{Name: name, Email: email, When: time.Now()}
	if _, err := wt.Commit(message, &gogit.CommitOptions{Author: sig, Committer: sig}); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	if err := repo.Push(&gogit.PushOptions{
		Auth:     auth,
		RefSpecs: []config.RefSpec{config.RefSpec("+HEAD:refs/heads/" + branch)},
	}); err != nil && err != gogit.NoErrAlreadyUpToDate {
		return false, fmt.Errorf("push HEAD:%s: %w", branch, err)
	}
	return true, nil
}

// StageCommitPushPath opens the clone at repoPath and delegates to
// StageCommitPush. Callers that only hold the worktree path (the Deployment
// deploy phase) use this.
func StageCommitPushPath(repoPath, addPath, message, branch string, auth transport.AuthMethod, name, email string) (bool, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", repoPath, err)
	}
	return StageCommitPush(repo, addPath, message, branch, auth, name, email)
}
