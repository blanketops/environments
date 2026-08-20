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
Package crossplane reconciles the Crossplane GitHub provider credential
secret — a cluster-scoped ExternalSecret consumed by the Crossplane GitHub
(Upjet) provider in crossplane-system. It is not owned by any CR and has no
teardown: the provider is a platform singleton, not a per-resource
lifecycle.
*/
package crossplane

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GitHubProviderSecretReconciler converges the cluster-scoped ExternalSecret
// backing the Crossplane GitHub (Upjet) provider's credentials in
// crossplane-system. Not owned by any CR — the provider is a platform
// singleton with no teardown.
type GitHubProviderSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

// NewGitHubProviderSecretReconciler constructs a
// GitHubProviderSecretReconciler targeting the given ESO store.
func NewGitHubProviderSecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *GitHubProviderSecretReconciler {
	return &GitHubProviderSecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

// Reconcile ensures the singleton github-upjet-creds ExternalSecret exists
// in crossplane-system.
func (r *GitHubProviderSecretReconciler) Reconcile(ctx context.Context) error {
	const (
		externalSecretName = "github-upjet-creds"
		namespace          = "crossplane-system"
	)

	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      externalSecretName,
				"namespace": namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "crossplane-github-provider",
				},
			},
			"spec": map[string]any{
				"refreshInterval": "10s",
				"secretStoreRef": map[string]any{
					"name": r.StoreName,
					"kind": r.StoreKind,
				},
				"target": map[string]any{
					"name":           "example-creds",
					"creationPolicy": "Owner",
					"template": map[string]any{
						"type":          "Opaque",
						"engineVersion": "v2",
						"data": map[string]any{
							"credentials": `{"token": "{{ .token }}"}`,
						},
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": "token",
						"remoteRef": map[string]any{
							"key": "/blanketops/crossplane/github/token",
						},
					},
				},
			},
		},
	}

	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, client.ObjectKey{Name: externalSecretName, Namespace: namespace}, &existing)

	if apierrors.IsNotFound(err) {
		r.Log.Info("creating ExternalSecret for GitHub Upjet provider",
			"secret", externalSecretName, "store", r.StoreName)
		return r.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existing.Object["spec"], desired.Object["spec"]) {
		existing.Object["spec"] = desired.Object["spec"]
		r.Log.Info("updating ExternalSecret for GitHub Upjet provider",
			"secret", externalSecretName, "store", r.StoreName)
		return r.Client.Update(ctx, &existing)
	}

	r.Log.V(1).Info("ExternalSecret for GitHub Upjet provider already up-to-date",
		"secret", externalSecretName)
	return nil
}
