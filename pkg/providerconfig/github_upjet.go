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

package secrets

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GitHubProviderConfigReconciler converges the Crossplane ProviderConfig
// that points the upjet-github provider at its credentials Secret.
type GitHubProviderConfigReconciler struct {
	Client client.Client
	Log    logr.Logger
}

// NewGitHubProviderConfigReconciler constructs a
// GitHubProviderConfigReconciler.
func NewGitHubProviderConfigReconciler(
	c client.Client,
	log logr.Logger,
) *GitHubProviderConfigReconciler {
	return &GitHubProviderConfigReconciler{
		Client: c,
		Log:    log,
	}
}

// Reconcile ensures the singleton upjet-github ProviderConfig exists,
// pointing at the credentials Secret produced by the ExternalSecret
// reconciler.
func (r *GitHubProviderConfigReconciler) Reconcile(ctx context.Context) error {
	const (
		name      = "github-upjet"
		namespace = "crossplane-system"

		// This secret is produced by the ExternalSecret reconciler
		secretName = "example-creds"
		secretKey  = "credentials"
	)

	desired := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "github.upbound.io/v1beta1",
			"kind":       "ProviderConfig",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "crossplane-github-provider",
				},
			},
			"spec": map[string]interface{}{
				"credentials": map[string]interface{}{
					"source": "Secret",
					"secretRef": map[string]interface{}{
						"name":      secretName,
						"namespace": namespace,
						"key":       secretKey,
					},
				},
			},
		},
	}

	desired.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "github.upbound.io",
		Version: "v1beta1",
		Kind:    "ProviderConfig",
	})

	key := client.ObjectKey{Name: name}
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, key, current)
	switch {
	case apierrors.IsNotFound(err):
		r.Log.Info("Creating GitHub ProviderConfig", "name", name)
		return r.Client.Create(ctx, desired)

	case err != nil:
		return fmt.Errorf("get ProviderConfig: %w", err)

	default:
		// Update spec only, keep resourceVersion for safe update
		desired.SetResourceVersion(current.GetResourceVersion())

		r.Log.Info("Updating GitHub ProviderConfig", "name", name)
		return r.Client.Update(ctx, desired)
	}
}
