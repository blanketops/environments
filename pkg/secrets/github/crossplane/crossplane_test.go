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

package crossplane

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	testutil "github.com/blanketops/environments/pkg/secrets/internal/testutil"
)

func TestGitHubProviderSecretReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t)).Build()
	r := NewGitHubProviderSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "github-upjet-creds", "crossplane-system")
	spec, _, _ := unstructured.NestedMap(es.Object, "spec")
	storeRef := spec["secretStoreRef"].(map[string]any)
	if storeRef["name"] != "vault-store" || storeRef["kind"] != "ClusterSecretStore" {
		t.Fatalf("unexpected secretStoreRef: %+v", storeRef)
	}
}

func TestGitHubProviderSecretReconciler_Reconcile_UpdatesOnDrift(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t)).Build()
	r := NewGitHubProviderSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}

	r2 := NewGitHubProviderSecretReconciler(c, logr.Discard(), "new-store", "ClusterSecretStore")
	if err := r2.Reconcile(context.Background()); err != nil {
		t.Fatalf("drift Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "github-upjet-creds", "crossplane-system")
	spec, _, _ := unstructured.NestedMap(es.Object, "spec")
	storeRef := spec["secretStoreRef"].(map[string]any)
	if storeRef["name"] != "new-store" {
		t.Fatalf("expected store name to be updated to new-store, got %+v", storeRef)
	}
}

func TestGitHubProviderSecretReconciler_Reconcile_NoopWhenUpToDate(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t)).Build()
	r := NewGitHubProviderSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := testutil.GetExternalSecret(t, c, "github-upjet-creds", "crossplane-system")

	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := testutil.GetExternalSecret(t, c, "github-upjet-creds", "crossplane-system")

	if before.GetResourceVersion() != after.GetResourceVersion() {
		t.Fatalf("expected no-op reconcile to leave resourceVersion unchanged: before=%s after=%s",
			before.GetResourceVersion(), after.GetResourceVersion())
	}
}
