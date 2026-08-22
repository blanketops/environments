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

package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
	"github.com/blanketops/environments/pkg/utils"
)

// newBareRemote creates a local bare git repo (standing in for a real
// remote — no network needed) and returns its filesystem path, usable
// directly as a git URL.
func newBareRemote(t *testing.T) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if out, err := utils.RunGit("", "init", "--bare", "-b", "master", remote); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	return remote
}

// commitFile clones remote into a scratch dir, writes name=content, commits,
// and pushes, returning the new commit SHA.
func commitFile(t *testing.T, remote, name, content string) string {
	t.Helper()
	scratch := t.TempDir()
	if out, err := utils.RunGit("", "clone", remote, scratch); err != nil {
		t.Fatalf("clone scratch: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(scratch, name), []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if out, err := utils.RunGit(scratch, "add", name); err != nil {
		t.Fatalf("add: %v: %s", err, out)
	}
	if out, err := utils.RunGit(scratch, "-c", "user.email=test@test", "-c", "user.name=test", "commit", "-m", "commit "+name); err != nil {
		t.Fatalf("commit: %v: %s", err, out)
	}
	if out, err := utils.RunGit(scratch, "push"); err != nil {
		t.Fatalf("push: %v: %s", err, out)
	}
	sha, err := utils.RunGit(scratch, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return sha
}

func TestEnsureLocalClone_FreshClone(t *testing.T) {
	remote := newBareRemote(t)
	sha := commitFile(t, remote, "a.txt", "v1")

	repoPath := filepath.Join(t.TempDir(), "workdir")
	if err := ensureLocalClone(repoPath, remote, sha, nil); err != nil {
		t.Fatalf("ensureLocalClone: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoPath, "a.txt"))
	if err != nil {
		t.Fatalf("read cloned file: %v", err)
	}
	if string(got) != "v1" {
		t.Fatalf("a.txt = %q, want %q", got, "v1")
	}

	head, err := utils.RunGit(repoPath, "rev-parse", "HEAD")
	if err != nil || head != sha {
		t.Fatalf("HEAD = %q, %v; want %q", head, err, sha)
	}
}

func TestEnsureLocalClone_ExistingCloneFetchesNewCommit(t *testing.T) {
	remote := newBareRemote(t)
	sha1 := commitFile(t, remote, "a.txt", "v1")

	repoPath := filepath.Join(t.TempDir(), "workdir")
	if err := ensureLocalClone(repoPath, remote, sha1, nil); err != nil {
		t.Fatalf("first ensureLocalClone: %v", err)
	}

	sha2 := commitFile(t, remote, "a.txt", "v2")

	if err := ensureLocalClone(repoPath, remote, sha2, nil); err != nil {
		t.Fatalf("second ensureLocalClone: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoPath, "a.txt"))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(got) != "v2" {
		t.Fatalf("a.txt = %q, want %q (fetch+checkout to the new commit should have happened)", got, "v2")
	}

	head, err := utils.RunGit(repoPath, "rev-parse", "HEAD")
	if err != nil || head != sha2 {
		t.Fatalf("HEAD = %q, %v; want %q", head, err, sha2)
	}
}

func TestEnsureLocalClone_EmptyRefUsesDefaultBranch(t *testing.T) {
	remote := newBareRemote(t)
	commitFile(t, remote, "a.txt", "on-master")

	repoPath := filepath.Join(t.TempDir(), "workdir")
	if err := ensureLocalClone(repoPath, remote, "", nil); err != nil {
		t.Fatalf("ensureLocalClone with empty ref: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoPath, "a.txt"))
	if err != nil || string(got) != "on-master" {
		t.Fatalf("a.txt = %q, %v; want %q from the default branch", got, err, "on-master")
	}
}

func newDeploymentTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(corev1): %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(appsv1): %v", err)
	}
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(environmentv1alpha1): %v", err)
	}
	if err := kustomizev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(kustomizev1): %v", err)
	}
	if err := sourcev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(sourcev1): %v", err)
	}
	return scheme
}

func TestResolveGitSSHEnv_WritesKeyMaterialAndCleansUp(t *testing.T) {
	scheme := newDeploymentTestScheme(t)
	cr := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy", Namespace: "default"},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy-flux-ssh", Namespace: "default"},
		Data: map[string][]byte{
			"identity":    []byte("fake-private-key"),
			"known_hosts": []byte("fake-known-hosts"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	m := &KustomizeStrategyProvider{Client: c, Scheme: scheme, Log: logr.Discard()}

	env, cleanup, err := m.resolveGitSSHEnv(context.Background(), cr)
	if err != nil {
		t.Fatalf("resolveGitSSHEnv: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("env = %v, want exactly one GIT_SSH_COMMAND entry", env)
	}

	// Extract the key path the env var points at and verify its content and
	// permissions before cleanup runs. Parses
	// "GIT_SSH_COMMAND=ssh -i <key> -o UserKnownHostsFile=<known_hosts> ...".
	var keyPath string
	fields := splitFields(env[0])
	for i, f := range fields {
		if f == "-i" && i+1 < len(fields) {
			keyPath = fields[i+1]
		}
	}
	if keyPath == "" {
		t.Fatalf("could not find -i <key> in %q", env[0])
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key material: %v", err)
	}
	if string(data) != "fake-private-key" {
		t.Fatalf("key content = %q, want %q", data, "fake-private-key")
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("key file perms = %o, want 0600", perm)
	}

	cleanup()
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("expected key file to be removed after cleanup, stat err = %v", err)
	}
}

func TestResolveGitSSHEnv_SecretNotFound(t *testing.T) {
	scheme := newDeploymentTestScheme(t)
	cr := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy", Namespace: "default"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := &KustomizeStrategyProvider{Client: c, Scheme: scheme, Log: logr.Discard()}

	_, _, err := m.resolveGitSSHEnv(context.Background(), cr)
	if err == nil {
		t.Fatal("expected an error when the flux-ssh secret does not exist")
	}
}

// splitFields is a tiny whitespace splitter — avoids pulling in strings just
// for one test helper's sake given the rest of the file doesn't need it.
func splitFields(s string) []string {
	var fields []string
	var cur []rune
	for _, r := range s {
		if r == ' ' {
			if len(cur) > 0 {
				fields = append(fields, string(cur))
				cur = nil
			}
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		fields = append(fields, string(cur))
	}
	return fields
}

// TestKustomizeStrategyProvider_Teardown_RemovesCommittedManifests is a
// regression test for Teardown/removeAndPush sharing CommitAndPush's
// missing-clone bug (fixed in ReconcileKustomization, but Teardown had no
// repoURL/ref to call ensureLocalClone with, since it was never called from
// anywhere). Seeds a remote via CommitAndPush, then verifies Teardown
// actually removes what was committed and pushes the removal — proving the
// local clone/checkout it now performs works end to end.
func TestKustomizeStrategyProvider_Teardown_RemovesCommittedManifests(t *testing.T) {
	scheme := newDeploymentTestScheme(t)
	cr := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "teardown-test-deploy", Namespace: "default"},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "teardown-test-deploy-flux-ssh", Namespace: "default"},
		Data: map[string][]byte{
			"identity":    []byte("fake-private-key"),
			"known_hosts": []byte("fake-known-hosts"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	m := &KustomizeStrategyProvider{Client: c, Scheme: scheme, Log: logr.Discard()}

	remote := newBareRemote(t)
	sha := commitFile(t, remote, "README.md", "seed")

	repoPath := filepath.Join(os.TempDir(), cr.Name+"-manifests")
	t.Cleanup(func() { _ = os.RemoveAll(repoPath) })

	di := &intent.DeploymentIntent{
		Namespace: "default",
		ServiceUnits: []serviceunitIntent.ServiceUnitIntent{
			{Name: "api", Image: "example/api:v1", Port: 8080, Size: 1},
		},
	}

	// Seed the remote with committed workload manifests, exactly as
	// ReconcileKustomization would.
	if err := ensureLocalClone(repoPath, remote, sha, nil); err != nil {
		t.Fatalf("ensureLocalClone (seed): %v", err)
	}
	if err := m.CommitAndPush(repoPath, di, "prod", nil); err != nil {
		t.Fatalf("CommitAndPush (seed): %v", err)
	}

	manifestPath := filepath.Join(repoPath, "overlays", "prod", "api-deployment.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected seeded manifest to exist before Teardown: %v", err)
	}

	if err := m.Teardown(context.Background(), cr, di, remote, "", "prod"); err != nil {
		t.Fatalf("Teardown: %v", err)
	}

	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("expected api-deployment.yaml to be removed after Teardown, stat err = %v", err)
	}

	// Verify the removal was actually pushed: clone fresh from remote and
	// check the file is gone there too, not just in the local working copy.
	freshClone := t.TempDir()
	if out, err := utils.RunGit("", "clone", remote, freshClone); err != nil {
		t.Fatalf("clone fresh: %v: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(freshClone, "overlays", "prod", "api-deployment.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected manifest removal to be pushed to the remote, stat err = %v", err)
	}
}
