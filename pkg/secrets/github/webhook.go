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
Package github reconciles GitHub-related secrets into Kubernetes, keeping
them in sync with the resolved contracts owned by GitHubEvent resources and
platform-level provider configuration.

This file owns the GitHub webhook secret — the ExternalSecret backing the
shared secret GitHub signs webhook deliveries with, scoped to a GitHubEvent
resource, and the teardown of both that ExternalSecret and its
ESO-materialized Secret when the GitHubEvent is deleted.
*/
package github

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	githubeventResolution "github.com/blanketops/environments/resolution/githubevent"
)

type GitHubWebhookSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

func NewGitHubWebhookSecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *GitHubWebhookSecretReconciler {
	return &GitHubWebhookSecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

func (r *GitHubWebhookSecretReconciler) Reconcile(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent) error {
	if resolved == nil || resolved.Event == nil || resolved.Spec == nil {
		return fmt.Errorf("nil ResolvedGitHubEvent (resolver bug)")
	}

	event := resolved.Event
	webhook := resolved.Spec.Webhook

	if webhook.SecretRef.Name == "" || webhook.SecretRef.Key == "" {
		return nil
	}

	secretName := webhook.SecretRef.Name
	secretKey := webhook.SecretRef.Key

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
				"refreshInterval": "10s",
				"secretStoreRef": map[string]any{
					"name": r.StoreName,
					"kind": r.StoreKind,
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

	if err := controllerutil.SetControllerReference(event, desired, r.Client.Scheme()); err != nil {
		return err
	}

	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(desired), &existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	r.Log.Info("creating ExternalSecret for GitHub webhook",
		"event", event.Name, "secret", secretName, "store", r.StoreName)
	return r.Client.Create(ctx, desired)
}

// Delete removes both the ExternalSecret and the Secret ESO materialized
// from it. See git.BuildGitSSHSecretReconciler.Delete for why the Secret
// must be deleted explicitly rather than left to the ExternalSecret's
// ownerReference GC cascade.
func (r *GitHubWebhookSecretReconciler) Delete(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent) error {
	if resolved == nil || resolved.Event == nil || resolved.Spec == nil {
		return nil
	}
	webhook := resolved.Spec.Webhook
	if webhook.SecretRef.Name == "" {
		return nil
	}
	namespace := resolved.Event.Namespace

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("external-secrets.io/v1")
	obj.SetKind("ExternalSecret")
	obj.SetName(webhook.SecretRef.Name)
	obj.SetNamespace(namespace)
	if err := r.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{}
	secret.SetName(webhook.SecretRef.Name)
	secret.SetNamespace(namespace)
	if err := r.Client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
