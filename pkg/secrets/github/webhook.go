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
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

type GitHubWebhookSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewGitHubWebhookSecretReconciler(c client.Client, log logr.Logger) *GitHubWebhookSecretReconciler {
	return &GitHubWebhookSecretReconciler{
		Client: c,
		Log:    log,
	}
}

// Reconcile ensures an ExternalSecret exists for the GitHub webhook shared secret.
//
// CONTRACT:
// - resolved contains authoritative webhook intent
// - Event is used only for ownership + namespace
func (r *GitHubWebhookSecretReconciler) Reconcile(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent) error {

	if resolved == nil || resolved.Event == nil || resolved.Spec == nil {
		return fmt.Errorf("nil ResolvedGitHubEvent (resolver bug)")
	}

	event := resolved.Event
	webhook := resolved.Spec.Webhook

	if webhook.SecretRef.Name == "" || webhook.SecretRef.Key == "" {
		// No webhook secret requested
		return nil
	}

	secretName := webhook.SecretRef.Name
	secretKey := webhook.SecretRef.Key

	// -------------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// -------------------------------------------------------------------------
	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": event.Namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/domain":  "githubevent",
					"blanketops.dev/event":   event.Name,
				},
			},
			"spec": map[string]any{
				"refreshInterval": "0s",
				"secretStoreRef": map[string]any{
					"name": "blanketops-environments-fake",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]any{
					"name": secretName,
					"template": map[string]any{
						"type": "Opaque",
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": secretKey,
						"remoteRef": map[string]any{
							"key": "/blanketops/github/webhook/secret",
						},
					},
				},
			},
		},
	}

	// OWN the ExternalSecret (never the Secret)
	if err := controllerutil.SetControllerReference(
		event,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Fetch existing
	// -------------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKeyFromObject(desired),
		&existing,
	)
	if err == nil {
		// Intentionally no drift reconciliation for webhook secrets
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	// -------------------------------------------------------------------------
	// Create
	// -------------------------------------------------------------------------
	r.Log.Info(
		"Creating ExternalSecret for GitHub webhook",
		"event", event.Name,
		"secret", secretName,
	)

	return r.Client.Create(ctx, desired)
}
