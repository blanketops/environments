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

package build

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	buildResolution "github.com/blanketops/environments/resolution/build/resolve"

	"github.com/blanketops/environments/pkg/secrets/internal/testutil"
)

func newResolvedBuildWithServiceAccount(name, namespace, secretName string) *buildResolution.ResolvedBuild {
	return &buildResolution.ResolvedBuild{
		Build: &environmentv1alpha1.Build{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "build-uid"},
		},
		Spec: &buildResolution.ResolvedBuildSpec{
			ServiceAccount: &buildResolution.ResolvedServiceAccount{Secret: secretName},
		},
	}
}

func TestBuildRegistryExternalSecretReconciler_Reconcile_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background(), nil); err != nil {
		t.Fatalf("nil build should no-op, got: %v", err)
	}
	if err := r.Reconcile(context.Background(), &buildResolution.ResolvedBuild{}); err != nil {
		t.Fatalf("nil Build should no-op, got: %v", err)
	}
	noSA := &buildResolution.ResolvedBuild{
		Build: &environmentv1alpha1.Build{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "default"}},
		Spec:  &buildResolution.ResolvedBuildSpec{},
	}
	if err := r.Reconcile(context.Background(), noSA); err != nil {
		t.Fatalf("nil ServiceAccount should no-op, got: %v", err)
	}
	empty := newResolvedBuildWithServiceAccount("b", "default", "")
	if err := r.Reconcile(context.Background(), empty); err != nil {
		t.Fatalf("empty secret name should no-op, got: %v", err)
	}
}

func TestBuildRegistryExternalSecretReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	build := newResolvedBuildWithServiceAccount("my-build", "default", "my-build-registry")

	if err := r.Reconcile(context.Background(), build); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "my-build-registry", "default")
	owners := es.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "my-build" {
		t.Fatalf("expected owner reference to Build, got %+v", owners)
	}
}

func TestBuildRegistryExternalSecretReconciler_Reconcile_NoopWhenAlreadyExists(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	build := newResolvedBuildWithServiceAccount("my-build", "default", "my-build-registry")

	if err := r.Reconcile(context.Background(), build); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := testutil.GetExternalSecret(t, c, "my-build-registry", "default")

	// Like GitHubWebhookSecretReconciler, this reconciler short-circuits on
	// any existing ExternalSecret rather than diffing spec, so a StoreName
	// change here does not propagate. This test documents current behavior.
	r2 := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "new-store", "ClusterSecretStore")
	if err := r2.Reconcile(context.Background(), build); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := testutil.GetExternalSecret(t, c, "my-build-registry", "default")

	if before.GetResourceVersion() != after.GetResourceVersion() {
		t.Fatalf("expected existing ExternalSecret to be left untouched: before=%s after=%s",
			before.GetResourceVersion(), after.GetResourceVersion())
	}
}

func TestBuildRegistryExternalSecretReconciler_Delete_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Delete(context.Background(), nil); err != nil {
		t.Fatalf("nil build should no-op, got: %v", err)
	}
	empty := newResolvedBuildWithServiceAccount("b", "default", "")
	if err := r.Delete(context.Background(), empty); err != nil {
		t.Fatalf("empty secret name should no-op, got: %v", err)
	}
}

func TestBuildRegistryExternalSecretReconciler_Delete_RemovesExternalSecretAndSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	build := newResolvedBuildWithServiceAccount("my-build", "default", "my-build-registry")

	if err := r.Reconcile(context.Background(), build); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "my-build-registry", Namespace: "default"}}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed Secret: %v", err)
	}

	if err := r.Delete(context.Background(), build); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if testutil.ExternalSecretExists(t, c, "my-build-registry", "default") {
		t.Fatal("expected ExternalSecret to be deleted")
	}
	if testutil.SecretExists(t, c, "my-build-registry", "default") {
		t.Fatal("expected Secret to be deleted")
	}
}

func TestBuildRegistryExternalSecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewBuildRegistryExternalSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	build := newResolvedBuildWithServiceAccount("my-build", "default", "my-build-registry")

	if err := r.Delete(context.Background(), build); err != nil {
		t.Fatalf("expected Delete on nonexistent objects to be a no-op, got: %v", err)
	}
}
