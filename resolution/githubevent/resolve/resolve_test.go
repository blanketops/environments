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

package resolve

import (
	"strings"
	"testing"

	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func evWithContract(raw string) *eventsv1alpha1.GitHubEvent {
	return &eventsv1alpha1.GitHubEvent{
		Spec: eventsv1alpha1.GitHubEventSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"repository":"x/y","eventType":"push"}`

func TestResolveGitHubEvent_Nil(t *testing.T) {
	if _, err := ResolveGitHubEvent(nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestResolveGitHubEvent_EmptyContract(t *testing.T) {
	ev := &eventsv1alpha1.GitHubEvent{}
	if _, err := ResolveGitHubEvent(ev); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveGitHubEvent_InvalidJSON(t *testing.T) {
	ev := evWithContract(`{not json`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveGitHubEvent_RepositoryMissing(t *testing.T) {
	ev := evWithContract(`{"eventType":"push"}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), `"repository"`) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestResolveGitHubEvent_EventTypeMissing(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y"}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), `"eventType"`) {
		t.Fatalf("expected eventType error, got %v", err)
	}
}

func TestResolveGitHubEvent_MinimalValid(t *testing.T) {
	ev := evWithContract(minimalValid)
	resolved, err := ResolveGitHubEvent(ev)
	if err != nil {
		t.Fatalf("ResolveGitHubEvent: %v", err)
	}
	if resolved.Spec.Repository != "x/y" || resolved.Spec.EventType != "push" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
	if resolved.Spec.Webhook.SecretRef.Name != "" {
		t.Fatal("expected zero-value Webhook when not declared")
	}
	if resolved.Event != ev {
		t.Fatal("expected resolved.Event to reference the original CR")
	}
}

func TestResolveGitHubEvent_OptionalFields(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","eventId":"evt-1","ref":"refs/heads/main","commitSHA":"abc123","actor":"neo"}`)
	resolved, err := ResolveGitHubEvent(ev)
	if err != nil {
		t.Fatalf("ResolveGitHubEvent: %v", err)
	}
	if resolved.Spec.EventID != "evt-1" || resolved.Spec.Ref != "refs/heads/main" ||
		resolved.Spec.CommitSHA != "abc123" || resolved.Spec.Actor != "neo" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
}

func TestResolveGitHubEvent_WebhookAbsent(t *testing.T) {
	ev := evWithContract(minimalValid)
	resolved, err := ResolveGitHubEvent(ev)
	if err != nil {
		t.Fatalf("ResolveGitHubEvent: %v", err)
	}
	if resolved.Spec.Webhook != (ResolvedWebhook{}) {
		t.Fatalf("expected zero-value webhook, got %+v", resolved.Spec.Webhook)
	}
}

func TestResolveGitHubEvent_WebhookWrongType(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":"not-an-object"}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), `"webhook" must be an object`) {
		t.Fatalf("expected webhook type error, got %v", err)
	}
}

func TestResolveGitHubEvent_WebhookSecretRefAbsent(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":{}}`)
	resolved, err := ResolveGitHubEvent(ev)
	if err != nil {
		t.Fatalf("ResolveGitHubEvent: %v", err)
	}
	if resolved.Spec.Webhook.SecretRef.Name != "" {
		t.Fatalf("expected zero-value SecretRef, got %+v", resolved.Spec.Webhook.SecretRef)
	}
}

func TestResolveGitHubEvent_WebhookSecretRefWrongType(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":{"secretRef":"not-an-object"}}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), `"webhook.secretRef" must be an object`) {
		t.Fatalf("expected secretRef type error, got %v", err)
	}
}

func TestResolveGitHubEvent_WebhookSecretRefMissingName(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":{"secretRef":{"key":"hmac"}}}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), "webhook.secretRef.name") {
		t.Fatalf("expected secretRef.name error, got %v", err)
	}
}

func TestResolveGitHubEvent_WebhookSecretRefMissingKey(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":{"secretRef":{"name":"webhook-secret"}}}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), "webhook.secretRef.key") {
		t.Fatalf("expected secretRef.key error, got %v", err)
	}
}

func TestResolveGitHubEvent_WebhookSecretRefValid(t *testing.T) {
	ev := evWithContract(`{"repository":"x/y","eventType":"push","webhook":{"secretRef":{"name":"webhook-secret","key":"hmac"}}}`)
	resolved, err := ResolveGitHubEvent(ev)
	if err != nil {
		t.Fatalf("ResolveGitHubEvent: %v", err)
	}
	if resolved.Spec.Webhook.SecretRef.Name != "webhook-secret" || resolved.Spec.Webhook.SecretRef.Key != "hmac" {
		t.Fatalf("unexpected SecretRef: %+v", resolved.Spec.Webhook.SecretRef)
	}
}

func TestResolveGitHubEvent_RepositoryWrongType(t *testing.T) {
	ev := evWithContract(`{"repository":5,"eventType":"push"}`)
	_, err := ResolveGitHubEvent(ev)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}
