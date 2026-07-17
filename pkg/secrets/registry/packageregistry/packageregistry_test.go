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

package packageregistry

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

func newResolvedPackageWithRegistry(name, namespace, secretName string) *packageResolution.ResolvedPackage {
	return &packageResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "package-uid"},
		},
		Spec: &packageResolution.ResolvedPackageSpec{
			PackageRepository: packageResolution.ResolvedPackageRepository{CredentialsSecret: secretName},
		},
	}
}

func TestPackageRegistrySecretReconciler_Reconcile_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background(), nil); err != nil {
		t.Fatalf("nil resolvedPackage should no-op, got: %v", err)
	}
	if err := r.Reconcile(context.Background(), &packageResolution.ResolvedPackage{}); err != nil {
		t.Fatalf("nil Package should no-op, got: %v", err)
	}
	empty := newResolvedPackageWithRegistry("pkg1", "default", "")
	if err := r.Reconcile(context.Background(), empty); err != nil {
		t.Fatalf("empty secret name should no-op, got: %v", err)
	}
}

func TestPackageRegistrySecretReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRegistry("pkg1", "default", "pkg1-registry")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "pkg1-registry", "default")
	owners := es.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "pkg1" {
		t.Fatalf("expected owner reference to Package, got %+v", owners)
	}
}

func TestPackageRegistrySecretReconciler_Reconcile_UpdatesOnDrift(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRegistry("pkg1", "default", "pkg1-registry")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}

	r2 := NewPackageRegistrySecretReconciler(c, logr.Discard(), "new-store", "ClusterSecretStore")
	if err := r2.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("drift Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "pkg1-registry", "default")
	spec, _, _ := unstructured.NestedMap(es.Object, "spec")
	storeRef := spec["secretStoreRef"].(map[string]any)
	if storeRef["name"] != "new-store" {
		t.Fatalf("expected store name to be updated to new-store, got %+v", storeRef)
	}
}

func TestPackageRegistrySecretReconciler_Reconcile_NoopWhenUpToDate(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRegistry("pkg1", "default", "pkg1-registry")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := testutil.GetExternalSecret(t, c, "pkg1-registry", "default")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := testutil.GetExternalSecret(t, c, "pkg1-registry", "default")

	if before.GetResourceVersion() != after.GetResourceVersion() {
		t.Fatalf("expected no-op reconcile to leave resourceVersion unchanged: before=%s after=%s",
			before.GetResourceVersion(), after.GetResourceVersion())
	}
}

func TestPackageRegistrySecretReconciler_Delete_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Delete(context.Background(), nil); err != nil {
		t.Fatalf("nil resolvedPackage should no-op, got: %v", err)
	}
	empty := newResolvedPackageWithRegistry("pkg1", "default", "")
	if err := r.Delete(context.Background(), empty); err != nil {
		t.Fatalf("empty secret name should no-op, got: %v", err)
	}
}

func TestPackageRegistrySecretReconciler_Delete_RemovesExternalSecretAndSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRegistry("pkg1", "default", "pkg1-registry")

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pkg1-registry", Namespace: "default"}}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed Secret: %v", err)
	}

	if err := r.Delete(context.Background(), pkg); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if testutil.ExternalSecretExists(t, c, "pkg1-registry", "default") {
		t.Fatal("expected ExternalSecret to be deleted")
	}
	if testutil.SecretExists(t, c, "pkg1-registry", "default") {
		t.Fatal("expected Secret to be deleted")
	}
}

func TestPackageRegistrySecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewPackageRegistrySecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	pkg := newResolvedPackageWithRegistry("pkg1", "default", "pkg1-registry")

	if err := r.Delete(context.Background(), pkg); err != nil {
		t.Fatalf("expected Delete on nonexistent objects to be a no-op, got: %v", err)
	}
}
