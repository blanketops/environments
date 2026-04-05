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

package github

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GitHubProviderSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewGitHubProviderSecretReconciler(
	c client.Client,
	log logr.Logger,
) *GitHubProviderSecretReconciler {
	return &GitHubProviderSecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *GitHubProviderSecretReconciler) Reconcile(
	ctx context.Context,
) error {

	const (
		externalSecretName = "github-upjet-creds"
		namespace          = "crossplane-system"
	)

	// -------------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// -------------------------------------------------------------------------
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
				"refreshInterval": "0s",
				"secretStoreRef": map[string]any{
					"name": "blanketops-environments-fake",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]any{
					"name":           "example-creds",
					"creationPolicy": "Owner",
					"template": map[string]any{
						"type":          "Opaque",
						"engineVersion": "v2",
						"data": map[string]any{
							"credentials": `{
  "token": "{{ .token }}"
}`,
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

	// -------------------------------------------------------------------------
	// Fetch existing
	// -------------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      externalSecretName,
			Namespace: namespace,
		},
		&existing,
	)

	// -------------------------------------------------------------------------
	// Create
	// -------------------------------------------------------------------------
	if apierrors.IsNotFound(err) {
		r.Log.Info(
			"Creating ExternalSecret for GitHub Upjet provider",
			"secret", externalSecretName,
		)
		return r.Client.Create(ctx, desired)
	}

	if err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Update (spec drift only)
	// -------------------------------------------------------------------------
	if !reflect.DeepEqual(
		existing.Object["spec"],
		desired.Object["spec"],
	) {
		existing.Object["spec"] = desired.Object["spec"]

		r.Log.Info(
			"Updating ExternalSecret for GitHub Upjet provider",
			"secret", externalSecretName,
		)

		return r.Client.Update(ctx, &existing)
	}

	// -------------------------------------------------------------------------
	// No-op
	// -------------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret for GitHub Upjet provider already up-to-date",
		"secret", externalSecretName,
	)

	return nil
}
