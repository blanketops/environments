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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return scheme
}

func newResolvedEnvironment(name, namespace string, contract *environmentresolve.ResolvedEnvironmentContract) *environmentresolve.ResolvedEnvironment {
	return &environmentresolve.ResolvedEnvironment{
		Environment: &environmentv1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "env-uid"},
		},
		Spec: &environmentresolve.ResolvedEnvironmentSpec{
			Contract: contract,
		},
	}
}

func getStore(t *testing.T, c client.Client, kind, name, namespace string) *unstructured.Unstructured {
	t.Helper()
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("external-secrets.io/v1")
	u.SetKind(kind)
	if err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, u); err != nil {
		t.Fatalf("get %s %s/%s: %v", kind, namespace, name, err)
	}
	return u
}

func storeExists(t *testing.T, c client.Client, kind, name, namespace string) bool {
	t.Helper()
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("external-secrets.io/v1")
	u.SetKind(kind)
	err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, u)
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("get %s %s/%s: %v", kind, namespace, name, err)
	return false
}

func vaultContract() *environmentresolve.ResolvedEnvironmentContract {
	return &environmentresolve.ResolvedEnvironmentContract{
		SecretStoreProvider: "vault",
		Vault: &environmentresolve.ResolvedVaultConfig{
			Address:   "https://vault:8200",
			Path:      "secret",
			Role:      "environments",
			MountPath: "kubernetes",
			Version:   "v2",
		},
	}
}

func TestSecretStoreReconciler_Reconcile_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())

	cases := []*environmentresolve.ResolvedEnvironment{
		nil,
		{},
		{Environment: &environmentv1alpha1.Environment{}},
		newResolvedEnvironment("env1", "default", nil), // no contract
	}
	for _, resolved := range cases {
		if err := r.Reconcile(context.Background(), resolved, "ClusterSecretStore"); err != nil {
			t.Fatalf("expected no-op (nil error) for %+v, got: %v", resolved, err)
		}
	}
}

func TestSecretStoreReconciler_Reconcile_RejectsUnrecognizedKind(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "default", vaultContract())

	if err := r.Reconcile(context.Background(), env, "NotAKind"); err == nil {
		t.Fatal("expected error for unrecognized store kind")
	}
}

func TestSecretStoreReconciler_Reconcile_NoopWhenProviderUnspecified(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "default", &environmentresolve.ResolvedEnvironmentContract{})

	if err := r.Reconcile(context.Background(), env, "ClusterSecretStore"); err != nil {
		t.Fatalf("expected no-op for empty provider, got: %v", err)
	}
}

func TestSecretStoreReconciler_Reconcile_CreatesVaultClusterSecretStore(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "default", vaultContract())

	if err := r.Reconcile(context.Background(), env, "ClusterSecretStore"); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	store := getStore(t, c, "ClusterSecretStore", StoreVault, "")
	if len(store.GetOwnerReferences()) != 0 {
		t.Fatalf("expected no owner references on cluster-scoped store, got %+v", store.GetOwnerReferences())
	}
	server, _, _ := unstructured.NestedString(store.Object, "spec", "provider", "vault", "server")
	if server != "https://vault:8200" {
		t.Fatalf("unexpected server: %q", server)
	}
	role, _, _ := unstructured.NestedString(store.Object, "spec", "provider", "vault", "auth", "kubernetes", "role")
	if role != "environments" {
		t.Fatalf("unexpected role: %q", role)
	}
}

func TestSecretStoreReconciler_Reconcile_CreatesSecretStoreWithOwnerRef(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "team-a", vaultContract())

	if err := r.Reconcile(context.Background(), env, "SecretStore"); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	store := getStore(t, c, "SecretStore", StoreVault, "team-a")
	owners := store.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "env1" {
		t.Fatalf("expected owner reference to Environment env1, got %+v", owners)
	}
}

func TestSecretStoreReconciler_Reconcile_SetsServiceAccountRefWhenDeclared(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	contract := vaultContract()
	contract.Vault.ServiceAccountName = "external-secrets"
	env := newResolvedEnvironment("env1", "default", contract)

	if err := r.Reconcile(context.Background(), env, "ClusterSecretStore"); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	store := getStore(t, c, "ClusterSecretStore", StoreVault, "")
	saName, _, _ := unstructured.NestedString(store.Object, "spec", "provider", "vault", "auth", "kubernetes", "serviceAccountRef", "name")
	if saName != "external-secrets" {
		t.Fatalf("unexpected serviceAccountRef.name: %q", saName)
	}
}

// This is the behavior the design question in review surfaced: unlike the
// composed-CR secret reconcilers under pkg/secrets (create-once, never
// revisit), this reconciler always converges the store to the *current*
// declared config — a second Environment with different Vault values
// updates the shared ClusterSecretStore rather than being silently ignored.
func TestSecretStoreReconciler_Reconcile_ConvergesOnSubsequentCalls(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "default", vaultContract())

	if err := r.Reconcile(context.Background(), env, "ClusterSecretStore"); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}

	updated := vaultContract()
	updated.Vault.Address = "https://different-vault:8200"
	env2 := newResolvedEnvironment("env1", "default", updated)
	if err := r.Reconcile(context.Background(), env2, "ClusterSecretStore"); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}

	store := getStore(t, c, "ClusterSecretStore", StoreVault, "")
	server, _, _ := unstructured.NestedString(store.Object, "spec", "provider", "vault", "server")
	if server != "https://different-vault:8200" {
		t.Fatalf("expected store to converge to updated address, got %q", server)
	}
}

func TestSecretStoreReconciler_Reconcile_UnimplementedProvidersError(t *testing.T) {
	cases := []struct {
		provider string
		storeKey string
	}{
		{"aws", "aws"},
		{"azure", "azure"},
		{"gcp", "gcp"},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
			r := NewSecretStoreReconciler(c, logr.Discard())
			env := newResolvedEnvironment("env1", "default", &environmentresolve.ResolvedEnvironmentContract{
				SecretStoreProvider: tc.provider,
			})

			err := r.Reconcile(context.Background(), env, "ClusterSecretStore")
			if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
				t.Fatalf("expected not-yet-implemented error for provider %q, got: %v", tc.provider, err)
			}
		})
	}
}

func TestSecretStoreReconciler_Reconcile_VaultProviderMissingConfigErrors(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	r := NewSecretStoreReconciler(c, logr.Discard())
	env := newResolvedEnvironment("env1", "default", &environmentresolve.ResolvedEnvironmentContract{
		SecretStoreProvider: "vault",
		// Vault left nil — resolve.go should never produce this, but the
		// reconciler must not silently apply a broken store if it does.
	})

	if err := r.Reconcile(context.Background(), env, "ClusterSecretStore"); err == nil {
		t.Fatal("expected error when provider is vault but Vault config is nil")
	}
	if storeExists(t, c, "ClusterSecretStore", StoreVault, "") {
		t.Fatal("expected no store to be created")
	}
}
