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

package hookurl

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	testutil "github.com/blanketops/environments/pkg/secrets/internal/testutil"
	gitrepoResolution "github.com/blanketops/environments/resolution/gitrepository"
)

func newResolvedGitRepository(name, namespace, hookURL string) *gitrepoResolution.ResolvedGitRepository {
	return &gitrepoResolution.ResolvedGitRepository{
		Repository: &sourcesv1alpha1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "repo-uid"},
		},
		Spec: &gitrepoResolution.ResolvedGitRepositorySpec{HookURL: hookURL},
	}
}

// seedHookURLSecret creates the hookurl Secret directly with Data populated,
// simulating the API server's StringData->Data conversion on persist — a
// step the fake client does not perform.
func seedHookURLSecret(t *testing.T, c client.Client, repoName, namespace, hookURL string) *corev1.Secret {
	t.Helper()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: HookURLSecretName(repoName), Namespace: namespace},
		Data:       map[string][]byte{HookURLSecretKey: []byte(hookURL)},
	}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed hookurl secret: %v", err)
	}
	return secret
}

func TestHookURLSecretName(t *testing.T) {
	if got, want := HookURLSecretName("my-repo"), "my-repo-hookurl"; got != want {
		t.Fatalf("HookURLSecretName: got %q want %q", got, want)
	}
}

func TestHookURLSecretReconciler_Reconcile_ErrorsOnIncompleteInput(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())

	cases := []*gitrepoResolution.ResolvedGitRepository{
		nil,
		{},
		{Repository: &sourcesv1alpha1.GitRepository{}},
	}
	for _, resolved := range cases {
		if err := r.Reconcile(context.Background(), resolved); err == nil {
			t.Fatalf("expected error for incomplete input %+v", resolved)
		}
	}
}

func TestHookURLSecretReconciler_Reconcile_ErrorsOnEmptyHookURL(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())
	repo := newResolvedGitRepository("my-repo", "default", "")

	err := r.Reconcile(context.Background(), repo)
	if err == nil || !strings.Contains(err.Error(), "resolver bug") {
		t.Fatalf("expected resolver-bug error for empty hookUrl, got: %v", err)
	}
}

func TestHookURLSecretReconciler_Reconcile_Creates(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())
	repo := newResolvedGitRepository("my-repo", "default", "https://hooks.example.com/my-repo")

	if err := r.Reconcile(context.Background(), repo); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-repo-hookurl", Namespace: "default"}, secret); err != nil {
		t.Fatalf("get hookurl secret: %v", err)
	}
	// The fake client stores objects as given and, unlike a real API server,
	// never converts StringData into Data on write — assert on the field the
	// reconciler actually set.
	if secret.StringData[HookURLSecretKey] != "https://hooks.example.com/my-repo" {
		t.Fatalf("unexpected hookurl secret data: %+v", secret.StringData)
	}
	owners := secret.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "my-repo" {
		t.Fatalf("expected owner reference to GitRepository, got %+v", owners)
	}
}

func TestHookURLSecretReconciler_Reconcile_UpdatesOnDrift(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())

	// Seed the secret with Data populated directly (rather than going
	// through Reconcile) to simulate the API server's StringData->Data
	// conversion on persist, which the fake client does not perform.
	seedHookURLSecret(t, c, "my-repo", "default", "https://hooks.example.com/my-repo")

	updated := newResolvedGitRepository("my-repo", "default", "https://hooks.example.com/my-repo-new")
	if err := r.Reconcile(context.Background(), updated); err != nil {
		t.Fatalf("drift Reconcile: %v", err)
	}

	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-repo-hookurl", Namespace: "default"}, secret); err != nil {
		t.Fatalf("get hookurl secret: %v", err)
	}
	if secret.StringData[HookURLSecretKey] != "https://hooks.example.com/my-repo-new" {
		t.Fatalf("expected hookUrl to be updated, got: %+v", secret.StringData)
	}
}

func TestHookURLSecretReconciler_Reconcile_NoopWhenUpToDate(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())
	repo := newResolvedGitRepository("my-repo", "default", "https://hooks.example.com/my-repo")

	before := seedHookURLSecret(t, c, "my-repo", "default", "https://hooks.example.com/my-repo")

	if err := r.Reconcile(context.Background(), repo); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	after := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-repo-hookurl", Namespace: "default"}, after); err != nil {
		t.Fatalf("get hookurl secret: %v", err)
	}

	if before.ResourceVersion != after.ResourceVersion {
		t.Fatalf("expected no-op reconcile to leave resourceVersion unchanged: before=%s after=%s",
			before.ResourceVersion, after.ResourceVersion)
	}
}

func TestHookURLSecretReconciler_Delete_NilRepo(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())

	if err := r.Delete(context.Background(), nil); err != nil {
		t.Fatalf("nil repo should no-op, got: %v", err)
	}
}

func TestHookURLSecretReconciler_Delete_RemovesSecret(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())
	repo := newResolvedGitRepository("my-repo", "default", "https://hooks.example.com/my-repo")

	if err := r.Reconcile(context.Background(), repo); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if err := r.Delete(context.Background(), repo); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if testutil.SecretExists(t, c, "my-repo-hookurl", "default") {
		t.Fatal("expected hookurl secret to be deleted")
	}
}

func TestHookURLSecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	scheme := testutil.NewScheme(t, sourcesv1alpha1.AddToScheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewHookURLSecretReconciler(c, scheme, logr.Discard())
	repo := newResolvedGitRepository("my-repo", "default", "https://hooks.example.com/my-repo")

	if err := r.Delete(context.Background(), repo); err != nil {
		t.Fatalf("expected Delete on nonexistent secret to be a no-op, got: %v", err)
	}
}
