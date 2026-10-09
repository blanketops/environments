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

package packagerepo

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"

	"github.com/blanketops/environments/pkg/secrets/internal/testutil"
)

func newResolvedPackageWithRepository(name, namespace, secretName string) *packageResolution.ResolvedPackage {
	return &packageResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "package-uid"},
		},
		Spec: &packageResolution.ResolvedPackageSpec{
			PackageRepository: packageResolution.ResolvedPackageRepository{CredentialsSecret: secretName},
		},
	}
}

func TestPackageRepositorySecretReconciler_Reconcile_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background(), nil); err != nil {
		t.Fatalf("nil resolvedPackage should no-op, got: %v", err)
	}
	if err := r.Reconcile(context.Background(), &packageResolution.ResolvedPackage{}); err != nil {
		t.Fatalf("nil Package should no-op, got: %v", err)
	}

	empty := newResolvedPackageWithRepository("pkg1", "default", "")
	if err := r.Reconcile(context.Background(), empty); err != nil {
		t.Fatalf("empty CredentialsSecret should no-op, got: %v", err)
	}
}

func TestPackageRepositorySecretReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "pkg1-packages-repo", "default")
	owners := es.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "pkg1" {
		t.Fatalf("expected owner reference to Package, got %+v", owners)
	}
}

// TestPackageRepositorySecretReconciler_Reconcile_SecretHasTheKeysKappReads
// locks in the Secret's shape. kapp-controller reads ssh-privatekey and
// ssh-knownhosts for git over SSH; with any other key name the fetch fails
// host key verification.
func TestPackageRepositorySecretReconciler_Reconcile_SecretHasTheKeysKappReads(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "pkg1-packages-repo", "default")
	data, found, err := unstructured.NestedStringMap(es.Object, "spec", "target", "template", "data")
	if err != nil || !found {
		t.Fatalf("target template data: found=%v err=%v", found, err)
	}
	if len(data) != 2 || data["ssh-privatekey"] == "" || data["ssh-knownhosts"] == "" {
		t.Errorf("template data = %v, want exactly ssh-privatekey and ssh-knownhosts", data)
	}

	remote, _, _ := unstructured.NestedSlice(es.Object, "spec", "data")
	keys := map[string]bool{}
	for _, item := range remote {
		key, _, _ := unstructured.NestedString(item.(map[string]any), "remoteRef", "key")
		keys[key] = true
	}
	if len(keys) != 2 || !keys["/blanketops/git/ssh-privatekey"] || !keys["/blanketops/git/known-hosts"] {
		t.Errorf("store keys = %v, want the git private key and known hosts", keys)
	}
}

func TestPackageRepositorySecretReconciler_Reconcile_UpdatesOnDrift(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}

	r2 := NewPackageRepositorySecretReconciler(c, logr.Discard(), "new-store", "ClusterSecretStore")
	if err := r2.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("drift Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "pkg1-packages-repo", "default")
	spec, _, _ := unstructured.NestedMap(es.Object, "spec")
	storeRef := spec["secretStoreRef"].(map[string]any)
	if storeRef["name"] != "new-store" {
		t.Fatalf("expected store name to be updated to new-store, got %+v", storeRef)
	}
}

func TestPackageRepositorySecretReconciler_Reconcile_NoopWhenUpToDate(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := testutil.GetExternalSecret(t, c, "pkg1-packages-repo", "default")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := testutil.GetExternalSecret(t, c, "pkg1-packages-repo", "default")

	if before.GetResourceVersion() != after.GetResourceVersion() {
		t.Fatalf("expected no-op reconcile to leave resourceVersion unchanged: before=%s after=%s",
			before.GetResourceVersion(), after.GetResourceVersion())
	}
}

func TestPackageRepositorySecretReconciler_Delete_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Delete(context.Background(), nil); err != nil {
		t.Fatalf("nil resolvedPackage should no-op, got: %v", err)
	}
	empty := newResolvedPackageWithRepository("pkg1", "default", "")
	if err := r.Delete(context.Background(), empty); err != nil {
		t.Fatalf("empty CredentialsSecret should no-op, got: %v", err)
	}
}

func TestPackageRepositorySecretReconciler_Delete_RemovesExternalSecretAndSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pkg1-packages-repo", Namespace: "default"}}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed Secret: %v", err)
	}

	if err := r.Delete(context.Background(), pkg); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if testutil.ExternalSecretExists(t, c, "pkg1-packages-repo", "default") {
		t.Fatal("expected ExternalSecret to be deleted")
	}
	if testutil.SecretExists(t, c, "pkg1-packages-repo", "default") {
		t.Fatal("expected Secret to be deleted")
	}
}

func TestPackageRepositorySecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRepositorySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRepository("pkg1", "default", "pkg1-packages-repo")

	if err := r.Delete(context.Background(), pkg); err != nil {
		t.Fatalf("expected Delete on nonexistent objects to be a no-op, got: %v", err)
	}
}
