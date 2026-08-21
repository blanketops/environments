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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/deployment/render/builders"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	"github.com/blanketops/environments/pkg/utils"
)

const fieldManager = "blanketops-kustomize-provider"
const defaultGitBranch = "master"

// KustomizeStrategyProvider implements the GitOps deployment path: it
// renders manifests, commits them to a repo, and ensures the Flux
// GitRepository/Kustomization CRs that make the cluster reconcile them.
type KustomizeStrategyProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewKustomizeStrategyProvider constructs a KustomizeStrategyProvider.
func NewKustomizeStrategyProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
) *KustomizeStrategyProvider {
	return &KustomizeStrategyProvider{
		Client: c,
		Scheme: scheme,
		Log:    log.WithName("kustomize-provider"),
	}
}

// BuildKustomize renders the Deployment and Service objects for every
// ServiceUnit in intent, without writing them anywhere.
func (m *KustomizeStrategyProvider) BuildKustomize(
	intent *intent.DeploymentIntent,
) ([]runtime.Object, error) {

	var objects []runtime.Object

	for _, su := range intent.ServiceUnits {

		deploy := builders.BuildDeployment(intent, &su)
		svc := builders.BuildService(intent, &su)

		objects = append(objects, deploy, svc)
	}

	return objects, nil
}

//
// PUBLIC ENTRYPOINT
//

// ReconcileKustomization is the GitOps path's public entrypoint: it renders
// and commits manifests to repoURL, then ensures the Flux GitRepository and
// Kustomization CRs that make the cluster apply them from ref/path.
func (m *KustomizeStrategyProvider) ReconcileKustomization(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
	intent *intent.DeploymentIntent,
	repoURL string,
	ref string,
	path string,
) error {

	if cr == nil {
		return fmt.Errorf("nil Deployment CR")
	}
	if repoURL == "" {
		return fmt.Errorf("repoURL required")
	}
	if path == "" {
		return fmt.Errorf("path required")
	}

	log := m.Log.WithValues(
		"deployment", cr.Name,
		"namespace", cr.Namespace,
	)

	log.Info("Reconciling GitOps resources")

	repoPath := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("%s-manifests", cr.Name),
	)

	env, ok := cr.Labels["environments.blanketops.dev/type"]
	if !ok || env == "" {
		return fmt.Errorf("deployment CR must define label environments.blanketops.dev/type")
	}

	// 0️⃣ Ensure repoPath is a local clone of repoURL at ref, authenticated
	// with the same deploy key Flux uses. Nothing else in this method's
	// call chain ever creates repoPath — every git operation below depends
	// on it already existing as a working clone.
	sshEnv, cleanup, err := m.resolveGitSSHEnv(ctx, cr)
	if err != nil {
		return fmt.Errorf("resolve git ssh credentials: %w", err)
	}
	defer cleanup()

	if err := ensureLocalClone(repoPath, repoURL, ref, sshEnv); err != nil {
		return fmt.Errorf("ensure local clone: %w", err)
	}

	// 1️⃣ Render + Commit
	if err := m.CommitAndPush(repoPath, intent, env, sshEnv); err != nil {
		return fmt.Errorf("commit and push failed: %w", err)
	}

	// 2️⃣ Ensure GitRepository
	if err := m.ensureGitRepository(ctx, cr, repoURL, ref); err != nil {
		return fmt.Errorf("failed ensuring GitRepository: %w", err)
	}

	// 3️⃣ Ensure Kustomization
	if err := m.ensureKustomization(ctx, cr, path); err != nil {
		return fmt.Errorf("failed ensuring Kustomization: %w", err)
	}

	log.Info("GitOps reconciliation complete")

	return nil
}

//
// INTERNALS
//

// CommitAndPush renders each ServiceUnit's Deployment/Service manifests
// into the env overlay under repoPath, removing previously generated
// workload files first, then commits and pushes the result. repoPath must
// already be a local clone (see ensureLocalClone); sshEnv authenticates
// the push.
func (m *KustomizeStrategyProvider) CommitAndPush(
	repoPath string,
	intent *intent.DeploymentIntent,
	env string,
	sshEnv []string,
) error {

	if env == "" {
		return fmt.Errorf("environment must not be empty")
	}

	overlayPath := filepath.Join(repoPath, "overlays", env)

	if err := os.MkdirAll(overlayPath, 0755); err != nil {
		return err
	}

	// -----------------------------------------
	// 1️⃣ Clean previously generated workload files
	// -----------------------------------------

	files, err := os.ReadDir(overlayPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		name := f.Name()

		if strings.HasSuffix(name, "-deployment.yaml") ||
			strings.HasSuffix(name, "-service.yaml") {

			if err := os.Remove(filepath.Join(overlayPath, name)); err != nil {
				return err
			}
		}
	}

	// -----------------------------------------
	// 2️⃣ Generate new workload manifests
	// -----------------------------------------

	var resourceEntries []string

	for _, su := range intent.ServiceUnits {

		deploy := builders.BuildDeployment(intent, &su)
		service := builders.BuildService(intent, &su)

		// Deployment
		deployYAML, err := m.renderObject(deploy)
		if err != nil {
			return err
		}

		deployFileName := fmt.Sprintf("%s-deployment.yaml", su.Name)
		deployFile := filepath.Join(overlayPath, deployFileName)

		if err := os.WriteFile(deployFile, deployYAML, 0644); err != nil {
			return err
		}

		// Service
		serviceYAML, err := m.renderObject(service)
		if err != nil {
			return err
		}

		serviceFileName := fmt.Sprintf("%s-service.yaml", su.Name)
		serviceFile := filepath.Join(overlayPath, serviceFileName)

		if err := os.WriteFile(serviceFile, serviceYAML, 0644); err != nil {
			return err
		}

		resourceEntries = append(resourceEntries,
			deployFileName,
			serviceFileName,
		)
	}

	// -----------------------------------------
	// 3️⃣ Regenerate kustomization.yaml (deterministic)
	// -----------------------------------------

	kustFile := filepath.Join(overlayPath, "kustomization.yaml")

	var kustContent bytes.Buffer

	kustContent.WriteString(
		`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base/manifests
`,
	)

	// Preserve static environment.yaml if present
	envFile := filepath.Join(overlayPath, "environment.yaml")
	if _, err := os.Stat(envFile); err == nil {
		kustContent.WriteString("  - environment.yaml\n")
	}

	for _, r := range resourceEntries {
		fmt.Fprintf(&kustContent, "  - %s\n", r)
	}

	if err := os.WriteFile(kustFile, kustContent.Bytes(), 0644); err != nil {
		return err
	}

	// -----------------------------------------
	// 4️⃣ Stage overlay directory
	// -----------------------------------------

	if _, err := utils.RunGit(repoPath, "add", filepath.Join("overlays", env)); err != nil {
		return err
	}

	// -----------------------------------------
	// 5️⃣ Commit only if changes exist
	// -----------------------------------------

	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath
	err = cmd.Run()

	if err == nil {
		m.Log.Info("No changes detected, skipping commit", "env", env)
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {

		if _, err := utils.RunGit(repoPath,
			"commit",
			"-m",
			fmt.Sprintf("render(%s): update workloads", env),
		); err != nil {
			return err
		}

		// ensureLocalClone leaves the working tree on a detached HEAD
		// (checking out FETCH_HEAD is never a branch), so a bare `push`
		// fails with "You are not currently on a branch" — the explicit
		// HEAD:<branch> refspec pushes whatever commit HEAD points at
		// regardless, landing it on the manifests repo's stable branch.
		if out, err := utils.RunGitWithEnv(repoPath, sshEnv, "push", "origin", "HEAD:"+defaultGitBranch); err != nil {
			return fmt.Errorf("push: %w: %s", err, out)
		}

		m.Log.Info("Changes committed and pushed", "env", env)
		return nil
	}

	return err
}

func (m *KustomizeStrategyProvider) renderObject(
	obj runtime.Object,
) ([]byte, error) {

	serializer := json.NewYAMLSerializer(
		json.DefaultMetaFactory,
		m.Scheme,
		m.Scheme,
	)

	var buffer bytes.Buffer

	if err := serializer.Encode(obj, &buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// resolveGitSSHEnv fetches cr's Flux deploy-key Secret (the same one
// ensureGitRepository validates and points Flux at) and materializes its
// private key and known_hosts as temp files, so this process can perform
// the same authenticated git operations Flux will later perform in-cluster.
// The caller must invoke the returned cleanup func to remove the temp files.
func (m *KustomizeStrategyProvider) resolveGitSSHEnv(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
) (env []string, cleanup func(), err error) {

	fluxSecret := fmt.Sprintf("%s-flux-ssh", cr.Name)

	var secret corev1.Secret
	if err := m.Client.Get(
		ctx,
		client.ObjectKey{Name: fluxSecret, Namespace: cr.Namespace},
		&secret,
	); err != nil {
		return nil, nil, fmt.Errorf(
			"flux git secret %q not found in namespace %q: %w",
			fluxSecret, cr.Namespace, err,
		)
	}

	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-git-ssh-", cr.Name))
	if err != nil {
		return nil, nil, fmt.Errorf("create ssh material dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	keyPath := filepath.Join(dir, "identity")
	if err := os.WriteFile(keyPath, secret.Data["identity"], 0600); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("write ssh key: %w", err)
	}

	knownHostsPath := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(knownHostsPath, secret.Data["known_hosts"], 0600); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("write known_hosts: %w", err)
	}

	sshCmd := fmt.Sprintf(
		"ssh -i %s -o UserKnownHostsFile=%s -o IdentitiesOnly=yes",
		keyPath, knownHostsPath,
	)
	return []string{"GIT_SSH_COMMAND=" + sshCmd}, cleanup, nil
}

// ensureLocalClone makes repoPath a local clone of repoURL checked out at
// ref (a commit SHA — see reconciliation.go's caller). Clones fresh if
// repoPath isn't a git repo yet, otherwise fetches and checks out in
// place. Using fetch+FETCH_HEAD rather than a branch/tag-specific command
// works uniformly whether ref is a branch, tag, or commit SHA. All network
// operations authenticate via env (see resolveGitSSHEnv).
func ensureLocalClone(repoPath, repoURL, ref string, env []string) error {
	if _, statErr := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}
		if out, err := utils.RunGitWithEnv("", env, "clone", repoURL, repoPath); err != nil {
			return fmt.Errorf("clone %s: %w: %s", repoURL, err, out)
		}
	} else if statErr != nil {
		return fmt.Errorf("stat repo path: %w", statErr)
	}

	checkoutRef := ref
	if checkoutRef == "" {
		checkoutRef = defaultGitBranch
	}
	if out, err := utils.RunGitWithEnv(repoPath, env, "fetch", "origin", checkoutRef); err != nil {
		return fmt.Errorf("fetch %s: %w: %s", checkoutRef, err, out)
	}
	if out, err := utils.RunGitWithEnv(repoPath, env, "checkout", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checkout %s: %w: %s", checkoutRef, err, out)
	}
	return nil
}

func (m *KustomizeStrategyProvider) ensureGitRepository(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
	repoURL string,
	ref string,
) error {

	fluxSecret := fmt.Sprintf("%s-flux-ssh", cr.Name)

	repo := &sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sourcev1.GroupVersion.String(),
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-source", cr.Name),
			Namespace: cr.Namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			Interval: metav1.Duration{Duration: 1 * time.Minute},
			URL:      repoURL,
			SecretRef: &fluxmeta.LocalObjectReference{
				Name: fluxSecret,
			},
		},
	}

	// ref is a commit SHA (see reconciliation.go's caller) — Commit takes
	// precedence over every other GitRepositoryRef field, so this pins Flux
	// to exactly the commit this reconcile resolved, not just a branch tip.
	if ref != "" {
		repo.Spec.Reference = &sourcev1.GitRepositoryRef{Commit: ref}
	} else {
		repo.Spec.Reference = &sourcev1.GitRepositoryRef{Branch: defaultGitBranch}
	}

	// Validate Flux secret exists (fail fast if infra broken)
	var existing corev1.Secret
	if err := m.Client.Get(
		ctx,
		client.ObjectKey{Name: fluxSecret, Namespace: cr.Namespace},
		&existing,
	); err != nil {
		return fmt.Errorf(
			"flux git secret %q not found in namespace %q: %w",
			fluxSecret,
			cr.Namespace,
			err,
		)
	}

	if err := controllerutil.SetControllerReference(cr, repo, m.Scheme); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	m.Log.Info("Applying GitRepository",
		"name", repo.Name,
		"secret", fluxSecret,
	)

	return m.Client.Patch(
		ctx,
		repo,
		client.Apply, //nolint:staticcheck
		&client.PatchOptions{FieldManager: fieldManager, Force: ptr.To(true)},
	)
}

func (m *KustomizeStrategyProvider) ensureKustomization(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
	path string,
) error {

	kust := &kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kustomizev1.GroupVersion.String(),
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kustomize", cr.Name),
			Namespace: cr.Namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			Interval:        metav1.Duration{Duration: 1 * time.Minute},
			Path:            path,
			Prune:           true,
			TargetNamespace: cr.Namespace,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: fmt.Sprintf("%s-source", cr.Name),
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, kust, m.Scheme); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	m.Log.Info("Applying Kustomization", "name", kust.Name)

	return m.Client.Patch(
		ctx,
		kust,
		client.Apply, //nolint:staticcheck
		&client.PatchOptions{FieldManager: fieldManager, Force: ptr.To(true)},
	)
}

// TeardownFluxResources deletes the Kustomization and GitRepository Flux
// CRs this Deployment's GitOps reconciliation created. It does not touch
// the manifests repo itself (no clone, no commit/push) — callers whose
// manifests-repo lifecycle is managed elsewhere (e.g. a mediator that
// provisions and deletes a whole per-Deployment repo via a Git host's API)
// call this alone rather than the fuller Teardown, since removing files
// from a repo that's about to be deleted wholesale is redundant at best and
// a failing clone/push at worst.
//
// The Kustomization is deleted before the GitRepository — stop reconciliation
// before the source it points at goes away, rather than after.
//
// GitRepository and Kustomization are ownerRef'd (SetControllerReference),
// so GC would eventually reclaim them — deleted explicitly here anyway for
// determinism.
func (m *KustomizeStrategyProvider) TeardownFluxResources(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
) error {
	kust := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kustomize", cr.Name),
			Namespace: cr.Namespace,
		},
	}
	if err := m.Client.Delete(ctx, kust); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete kustomization: %w", err)
	}

	repo := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-source", cr.Name),
			Namespace: cr.Namespace,
		},
	}
	if err := m.Client.Delete(ctx, repo); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete gitrepository: %w", err)
	}

	return nil
}

// Teardown removes GitOps-managed resources for this Deployment CR: deletes
// the workload manifests from the external Git repo (committed + pushed),
// then deletes the Kustomization and GitRepository Flux CRs via
// TeardownFluxResources. repoURL/ref identify the manifests repo exactly as
// ReconcileKustomization's caller does (ref is a commit SHA) — needed to
// ensure a local clone exists before removeAndPush can touch it.
//
// Only meaningful for a manifests repo whose lifecycle this type owns
// end to end (shared/persistent repo, files added and removed per
// Deployment). If something else owns that repo's lifecycle (creates and
// deletes the whole repo per Deployment), call TeardownFluxResources alone
// instead — see its doc comment.
//
// Order matters: the Flux CRs must be deleted (or their manifests must
// already be absent) before or alongside the repo cleanup — otherwise Flux
// may reconcile a stale kustomization.yaml pointing at files mid-removal.
// Deleting first, removing files second, is the safer order here since a
// missing Kustomization simply stops reconciling rather than reconciling
// a broken state.
func (m *KustomizeStrategyProvider) Teardown(
	ctx context.Context,
	cr *environmentv1alpha1.Deployment,
	intent *intent.DeploymentIntent,
	repoURL string,
	ref string,
	env string,
) error {
	log := m.Log.WithValues("deployment", cr.Name, "namespace", cr.Namespace)

	if err := m.TeardownFluxResources(ctx, cr); err != nil {
		return err
	}

	// Ensure a local clone exists, then remove committed manifests from the
	// external repo + push. Same clone/auth requirement as
	// ReconcileKustomization: nothing else guarantees repoPath is a working
	// git clone by the time we get here.
	repoPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-manifests", cr.Name))

	sshEnv, cleanup, err := m.resolveGitSSHEnv(ctx, cr)
	if err != nil {
		return fmt.Errorf("resolve git ssh credentials: %w", err)
	}
	defer cleanup()

	if err := ensureLocalClone(repoPath, repoURL, ref, sshEnv); err != nil {
		return fmt.Errorf("ensure local clone: %w", err)
	}

	if err := m.removeAndPush(repoPath, cr, intent, env, sshEnv); err != nil {
		return fmt.Errorf("remove manifests from gitops repo: %w", err)
	}

	log.Info("GitOps teardown complete")
	return nil
}

// removeAndPush removes cr's workload manifests from repoPath (already a
// local clone — see Teardown) and pushes the result. sshEnv authenticates
// the push.
func (m *KustomizeStrategyProvider) removeAndPush(
	repoPath string,
	cr *environmentv1alpha1.Deployment,
	intent *intent.DeploymentIntent,
	env string,
	sshEnv []string,
) error {
	overlayPath := filepath.Join(repoPath, "overlays", env)

	for _, su := range intent.ServiceUnits {
		for _, suffix := range []string{"-deployment.yaml", "-service.yaml"} {
			f := filepath.Join(overlayPath, su.Name+suffix)
			if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	// Regenerate kustomization.yaml without the removed entries.
	kustFile := filepath.Join(overlayPath, "kustomization.yaml")
	var kustContent bytes.Buffer
	kustContent.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n  - ../../base/manifests\n")
	envFile := filepath.Join(overlayPath, "environment.yaml")
	if _, err := os.Stat(envFile); err == nil {
		kustContent.WriteString("  - environment.yaml\n")
	}
	if err := os.WriteFile(kustFile, kustContent.Bytes(), 0644); err != nil {
		return err
	}

	if _, err := utils.RunGit(repoPath, "add", filepath.Join("overlays", env)); err != nil {
		return err
	}

	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		m.Log.Info("no manifest changes to remove", "env", env)
		return nil
	}
	if _, err := utils.RunGit(repoPath, "commit", "-m", fmt.Sprintf("render(%s): remove workloads for %s", env, cr.Name)); err != nil {
		return err
	}
	// See CommitAndPush's push call for why this uses an explicit
	// HEAD:<branch> refspec instead of a bare push.
	if out, err := utils.RunGitWithEnv(repoPath, sshEnv, "push", "origin", "HEAD:"+defaultGitBranch); err != nil {
		return fmt.Errorf("push: %w: %s", err, out)
	}
	return nil
}
