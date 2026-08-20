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

/*
Package api — secretstore.go provisions the ESO ClusterSecretStore or
SecretStore backing an Environment's declared secretStore.provider.

One reconciler, dispatched by provider string — vault.go/aws.go/azure.go/
gcp.go each contribute only the provider-specific spec.provider.<x> block;
the Get/apply/owner-ref orchestration below is identical across providers.

This reconciler owns the store *pointer* only. Secret values are populated
into the backend by a separate, out-of-band process; nothing here reads or
writes secret contents.
*/
package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

// SecretStoreReconciler converges the ESO ClusterSecretStore/SecretStore
// for an Environment's declared secretStore.provider to its current
// desired state.
type SecretStoreReconciler struct {
	Client client.Client
	Log    logr.Logger
}

// NewSecretStoreReconciler constructs a SecretStoreReconciler.
func NewSecretStoreReconciler(c client.Client, log logr.Logger) *SecretStoreReconciler {
	return &SecretStoreReconciler{Client: c, Log: log}
}

// Reconcile applies the ClusterSecretStore/SecretStore for the given
// resolved Environment's declared provider.
//
// Unlike the composed-CR secret reconcilers under pkg/secrets (which
// create once and never revisit an existing object), this always
// server-side-applies the desired spec — matching the convergence pattern
// used for actual workload resources (see
// pkg/apis/deployment/api.K8SProvider.applyDeployment/applyService). The
// store's connection config is expected to track whatever the Environment
// currently declares, and the core CQRS engine (core/command) already
// gates *when* this is invoked via each Domain's CanUpdate predicate — so
// by the time Reconcile runs, the caller has already decided the change is
// worth applying. This reconciler's job is only to converge, not to
// second-guess that decision by silently ignoring an existing store.
//
// storeKind must be StoreKindClusterSecretStore ("ClusterSecretStore") or
// StoreKindSecretStore ("SecretStore") — see pkg/apis/environment/query.
// ClusterSecretStore is cluster-scoped, so no owner reference is set on it
// (Kubernetes forbids a namespaced owner for a cluster-scoped object);
// SecretStore is namespaced to the Environment and owned by it.
//
// A provider without a builder yet (aws, azure, gcp — stubs pending their
// own ResolvedXConfig in resolution/environment/resolve) returns an error
// rather than applying an incomplete store.
func (r *SecretStoreReconciler) Reconcile(ctx context.Context, resolved *environmentresolve.ResolvedEnvironment, storeKind string) error {
	if resolved == nil || resolved.Environment == nil || resolved.Spec == nil || resolved.Spec.Contract == nil {
		return nil
	}
	if storeKind != "ClusterSecretStore" && storeKind != "SecretStore" {
		return fmt.Errorf("unrecognized secret store kind %q (expected \"ClusterSecretStore\" or \"SecretStore\")", storeKind)
	}

	contract := resolved.Spec.Contract
	providerBlock, storeName, err := buildProviderSpec(contract)
	if err != nil {
		return err
	}
	if providerBlock == nil {
		// Unrecognized/unspecified provider — nothing to provision.
		return nil
	}

	env := resolved.Environment

	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       storeKind,
			"metadata": map[string]any{
				"name": storeName,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/domain":  "environment",
				},
			},
			"spec": map[string]any{
				"provider": providerBlock,
			},
		},
	}

	if storeKind == "SecretStore" {
		desired.SetNamespace(env.Namespace)
		if err := controllerutil.SetControllerReference(env, desired, r.Client.Scheme()); err != nil {
			return err
		}
	}

	r.Log.Info("applying secret store", "kind", storeKind, "name", storeName,
		"provider", contract.SecretStoreProvider, "environment", env.Name)

	return r.Client.Patch(ctx, desired, client.Apply, //nolint:staticcheck
		&client.PatchOptions{FieldManager: "blanketops-environment-api", Force: ptr.To(true)})
}

// buildProviderSpec dispatches on the declared provider and returns the
// spec.provider.<x> block for it, plus the ClusterSecretStore/SecretStore
// name to use. A nil, nil return means the provider is unrecognized/empty
// and nothing should be provisioned.
func buildProviderSpec(contract *environmentresolve.ResolvedEnvironmentContract) (block map[string]any, storeName string, err error) {
	switch contract.SecretStoreProvider {
	case "vault", "SECRET_STORE_PROVIDER_VAULT":
		block, err := buildVaultProviderSpec(contract.Vault)
		return block, StoreVault, err
	case "aws", "SECRET_STORE_PROVIDER_AWS":
		block, err := buildAWSProviderSpec(contract)
		return block, StoreAWS, err
	case "azure", "SECRET_STORE_PROVIDER_AZURE":
		block, err := buildAzureProviderSpec(contract)
		return block, StoreAzure, err
	case "gcp", "SECRET_STORE_PROVIDER_GCP":
		block, err := buildGCPProviderSpec(contract)
		return block, StoreGCP, err
	default:
		return nil, "", nil
	}
}
