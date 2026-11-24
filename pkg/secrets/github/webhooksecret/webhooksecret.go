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
Package webhooksecret reconciles the per-repository webhook HMAC secret in the
GitRepository's namespace — an ExternalSecret (and the ESO-materialized Secret)
holding the shared secret GitHub signs webhook deliveries with. Upjet's
RepositoryWebhook consumes it via configuration.secretSecretRef so GitHub sends
an X-Hub-Signature header; the Argo Events github eventsource validates that
signature against the SAME value (it sources github-webhook-secret from the same
store key blanketops/environments/github/webhook/secret). Sourcing both from one
store key is what makes the signatures match — without this secret GitHub sends
no signature and a secret-configured eventsource rejects the delivery as
"missing signature".
*/
package webhooksecret

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gitrepoResolution "github.com/blanketops/environments/resolution/gitrepository/resolve"
)

// SecretKey is the key under which the HMAC secret is stored in the
// reconciled Secret's data. It matches the Argo Events eventsource's
// webhookSecret key so both derive identically from the store.
const SecretKey = "secret"

// remoteStoreKey is the store path the HMAC value is sourced from. It is the
// same key the GitHubEvent domain's webhook-secret ExternalSecret reads, so the
// GitHub hook secret and the eventsource's validation secret always match.
const remoteStoreKey = "blanketops/environments/github/webhook/secret"

// SecretName returns the per-repository webhook HMAC Secret name.
func SecretName(repoName string) string {
	return repoName + "-webhook-hmac"
}

// WebhookSecretReconciler converges the ExternalSecret backing the shared HMAC
// secret for a GitRepository's webhook, scoped to that GitRepository, and tears
// down both the ExternalSecret and its ESO-materialized Secret on deletion.
type WebhookSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

// NewWebhookSecretReconciler constructs a WebhookSecretReconciler targeting the
// given ESO store.
func NewWebhookSecretReconciler(c client.Client, log logr.Logger, storeName, storeKind string) *WebhookSecretReconciler {
	return &WebhookSecretReconciler{Client: c, Log: log, StoreName: storeName, StoreKind: storeKind}
}

// Reconcile creates the webhook-HMAC ExternalSecret for resolved's repository
// if it does not already exist.
func (r *WebhookSecretReconciler) Reconcile(ctx context.Context, resolved *gitrepoResolution.ResolvedGitRepository) error {
	if resolved == nil || resolved.Repository == nil || resolved.Spec == nil {
		return fmt.Errorf("resolved gitrepository is incomplete")
	}
	repo := resolved.Repository
	secretName := SecretName(repo.Name)

	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": repo.Namespace,
				"labels": map[string]any{
					"sources.blanketops.dev/gitrepository": repo.Name,
					"blanketops.dev/managed":               "true",
					"blanketops.dev/purpose":               "webhook-hmac",
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
						"secretKey": SecretKey,
						"remoteRef": map[string]any{
							"key":      remoteStoreKey,
							"property": "value",
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(repo, desired, r.Client.Scheme()); err != nil {
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

	r.Log.Info("creating ExternalSecret for GitHub webhook HMAC",
		"repository", repo.Name, "secret", secretName, "store", r.StoreName)
	return r.Client.Create(ctx, desired)
}

// Delete removes both the ExternalSecret and the Secret ESO materialized from
// it. The Secret is deleted explicitly rather than left to ownerReference GC,
// matching the other webhook-secret reconcilers.
func (r *WebhookSecretReconciler) Delete(ctx context.Context, resolved *gitrepoResolution.ResolvedGitRepository) error {
	if resolved == nil || resolved.Repository == nil {
		return nil
	}
	repo := resolved.Repository
	secretName := SecretName(repo.Name)

	es := &unstructured.Unstructured{}
	es.SetAPIVersion("external-secrets.io/v1")
	es.SetKind("ExternalSecret")
	es.SetName(secretName)
	es.SetNamespace(repo.Namespace)
	if err := r.Client.Delete(ctx, es); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{}
	secret.SetName(secretName)
	secret.SetNamespace(repo.Namespace)
	if err := r.Client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
