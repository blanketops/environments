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

package gitrepository

import (
	"context"
	"strings"
	"testing"

	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func repoWithContract(raw string) *sourcesv1alpha1.GitRepository {
	return &sourcesv1alpha1.GitRepository{
		Spec: sourcesv1alpha1.GitRepositorySpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"provider":"github","hookUrl":"https://hooks/x","repository":{"owner":"blanketops","name":"environments"}}`

func TestResolveGitRepository_Nil(t *testing.T) {
	if _, err := ResolveGitRepository(nil); err == nil {
		t.Fatal("expected error for nil repo")
	}
}

func TestResolveGitRepository_EmptyContract(t *testing.T) {
	r := &sourcesv1alpha1.GitRepository{}
	if _, err := ResolveGitRepository(r); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveGitRepository_InvalidJSON(t *testing.T) {
	r := repoWithContract(`{not json`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveGitRepository_ProviderMissing(t *testing.T) {
	r := repoWithContract(`{"hookUrl":"x","repository":{"owner":"o","name":"n"}}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), `"provider"`) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestResolveGitRepository_HookURLMissing(t *testing.T) {
	r := repoWithContract(`{"provider":"github","repository":{"owner":"o","name":"n"}}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), `"hookUrl"`) {
		t.Fatalf("expected hookUrl error, got %v", err)
	}
}

func TestResolveGitRepository_RepositoryMissing(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x"}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "repository is required") {
		t.Fatalf("expected repository-required error, got %v", err)
	}
}

func TestResolveGitRepository_RepositoryOwnerMissing(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x","repository":{"name":"n"}}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "repository.owner") {
		t.Fatalf("expected repository.owner error, got %v", err)
	}
}

func TestResolveGitRepository_RepositoryNameMissing(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x","repository":{"owner":"o"}}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "repository.name") {
		t.Fatalf("expected repository.name error, got %v", err)
	}
}

func TestResolveGitRepository_MinimalValid(t *testing.T) {
	r := repoWithContract(minimalValid)
	resolved, err := ResolveGitRepository(r)
	if err != nil {
		t.Fatalf("ResolveGitRepository: %v", err)
	}
	if resolved.Spec.Provider != "github" || resolved.Spec.HookURL != "https://hooks/x" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
	if resolved.Spec.Repository.Owner != "blanketops" || resolved.Spec.Repository.Name != "environments" {
		t.Fatalf("unexpected repository ref: %+v", resolved.Spec.Repository)
	}
	if resolved.Spec.Webhooks != nil {
		t.Fatal("expected nil Webhooks when not declared")
	}
}

func TestResolveGitRepository_WebhookEntryNotObject(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x","repository":{"owner":"o","name":"n"},"webhooks":["not-an-object"]}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "webhooks[0] must be an object") {
		t.Fatalf("expected webhooks[0] error, got %v", err)
	}
}

func TestResolveGitRepository_WebhookNoEvents_Skipped(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x","repository":{"owner":"o","name":"n"},"webhooks":[{},{"events":[]}]}`)
	resolved, err := ResolveGitRepository(r)
	if err != nil {
		t.Fatalf("ResolveGitRepository: %v", err)
	}
	if len(resolved.Spec.Webhooks) != 0 {
		t.Fatalf("expected webhooks with no valid events to be skipped, got %+v", resolved.Spec.Webhooks)
	}
}

func TestResolveGitRepository_WebhookEventsMixedValidInvalid(t *testing.T) {
	r := repoWithContract(`{"provider":"github","hookUrl":"x","repository":{"owner":"o","name":"n"},"webhooks":[{"events":["push","",123,"pull_request"]}]}`)
	resolved, err := ResolveGitRepository(r)
	if err != nil {
		t.Fatalf("ResolveGitRepository: %v", err)
	}
	if len(resolved.Spec.Webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(resolved.Spec.Webhooks))
	}
	if len(resolved.Spec.Webhooks[0].Events) != 2 {
		t.Fatalf("expected 2 valid events (empty/non-string skipped), got %+v", resolved.Spec.Webhooks[0].Events)
	}
}

func TestResolveGitRepository_ProviderWrongType(t *testing.T) {
	r := repoWithContract(`{"provider":5,"hookUrl":"x","repository":{"owner":"o","name":"n"}}`)
	_, err := ResolveGitRepository(r)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	r := repoWithContract(minimalValid)
	resolved, err := a.Resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Provider != "github" {
		t.Fatalf("unexpected Provider: %q", resolved.Spec.Provider)
	}
}
