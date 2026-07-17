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

package webhook

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	githubeventResolution "github.com/blanketops/environments/resolution/githubevent/resolve"

	"github.com/blanketops/environments/pkg/secrets/internal/testutil"
)

func newResolvedGitHubEvent(name, namespace, secretName, secretKey string) *githubeventResolution.ResolvedGitHubEvent {
	return &githubeventResolution.ResolvedGitHubEvent{
		Event: &eventsv1alpha1.GitHubEvent{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "event-uid"},
		},
		Spec: &githubeventResolution.ResolvedGitHubEventSpec{
			Webhook: githubeventResolution.ResolvedWebhook{
				SecretRef: githubeventResolution.ResolvedSecretRef{Name: secretName, Key: secretKey},
			},
		},
	}
}

func TestGitHubWebhookSecretReconciler_Reconcile_ErrorsOnIncompleteInput(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	cases := []*githubeventResolution.ResolvedGitHubEvent{
		nil,
		{},
		{Event: &eventsv1alpha1.GitHubEvent{}},
	}
	for _, resolved := range cases {
		if err := r.Reconcile(context.Background(), resolved); err == nil {
			t.Fatalf("expected error for incomplete input %+v", resolved)
		}
	}
}

func TestGitHubWebhookSecretReconciler_Reconcile_NoopWhenSecretRefEmpty(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	ev := newResolvedGitHubEvent("my-event", "default", "", "")

	if err := r.Reconcile(context.Background(), ev); err != nil {
		t.Fatalf("empty SecretRef should no-op, got: %v", err)
	}
}

func TestGitHubWebhookSecretReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	ev := newResolvedGitHubEvent("my-event", "default", "my-event-webhook", "secret")

	if err := r.Reconcile(context.Background(), ev); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	es := testutil.GetExternalSecret(t, c, "my-event-webhook", "default")
	owners := es.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "my-event" {
		t.Fatalf("expected owner reference to GitHubEvent, got %+v", owners)
	}
}

func TestGitHubWebhookSecretReconciler_Reconcile_NoopWhenAlreadyExists(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	ev := newResolvedGitHubEvent("my-event", "default", "my-event-webhook", "secret")

	if err := r.Reconcile(context.Background(), ev); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := testutil.GetExternalSecret(t, c, "my-event-webhook", "default")

	// This reconciler short-circuits on any existing ExternalSecret rather
	// than diffing spec — unlike the other reconcilers in pkg/secrets, it
	// never re-applies drift (e.g. a StoreName change here would not
	// propagate). This test documents that current behavior.
	r2 := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "new-store", "ClusterSecretStore")
	if err := r2.Reconcile(context.Background(), ev); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := testutil.GetExternalSecret(t, c, "my-event-webhook", "default")

	if before.GetResourceVersion() != after.GetResourceVersion() {
		t.Fatalf("expected existing ExternalSecret to be left untouched: before=%s after=%s",
			before.GetResourceVersion(), after.GetResourceVersion())
	}
}

func TestGitHubWebhookSecretReconciler_Delete_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")

	if err := r.Delete(context.Background(), nil); err != nil {
		t.Fatalf("nil resolved should no-op, got: %v", err)
	}
	empty := newResolvedGitHubEvent("my-event", "default", "", "")
	if err := r.Delete(context.Background(), empty); err != nil {
		t.Fatalf("empty SecretRef should no-op, got: %v", err)
	}
}

func TestGitHubWebhookSecretReconciler_Delete_RemovesExternalSecretAndSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	ev := newResolvedGitHubEvent("my-event", "default", "my-event-webhook", "secret")

	if err := r.Reconcile(context.Background(), ev); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "my-event-webhook", Namespace: "default"}}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed Secret: %v", err)
	}

	if err := r.Delete(context.Background(), ev); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if testutil.ExternalSecretExists(t, c, "my-event-webhook", "default") {
		t.Fatal("expected ExternalSecret to be deleted")
	}
	if testutil.SecretExists(t, c, "my-event-webhook", "default") {
		t.Fatal("expected Secret to be deleted")
	}
}

func TestGitHubWebhookSecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, eventsv1alpha1.AddToScheme)).Build()
	r := NewGitHubWebhookSecretReconciler(c, logr.Discard(), "vault-store", "ClusterSecretStore")
	ev := newResolvedGitHubEvent("my-event", "default", "my-event-webhook", "secret")

	if err := r.Delete(context.Background(), ev); err != nil {
		t.Fatalf("expected Delete on nonexistent objects to be a no-op, got: %v", err)
	}
}
