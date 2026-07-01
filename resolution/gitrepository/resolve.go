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
Package gitrepository implements resolution for the GitRepository CR.

The GitRepository CR stores its spec as a raw JSON contract (spec.contract).
ResolveGitRepository decodes this into a fully typed ResolvedGitRepository —
the authoritative runtime representation consumed by all downstream domain
and application logic.

Resolution validates required fields (provider, hookUrl, repository) and
normalises the optional webhook list. Webhook entries with no valid events
are silently skipped — an empty event list is not a valid subscription.

All failures surface as errors. Resolution never panics.
*/
package gitrepository

import (
	"encoding/json"
	"fmt"

	sourcesv1alpha1 "github.com/BlanketOps/environments-api/api/sources/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedGitRepository pairs the original Kubernetes GitRepository object
// with its fully decoded and validated spec.
type ResolvedGitRepository struct {
	Repository *sourcesv1alpha1.GitRepository
	Spec       *ResolvedGitRepositorySpec
}

// ResolvedGitRepositorySpec is the decoded GitRepository spec.
type ResolvedGitRepositorySpec struct {
	// Provider is the normalised provider string ("github", "gitlab", etc.).
	// The contract adapter maps this to the proto GitProvider wrapper message.
	Provider string
	// HookURL is the webhook delivery endpoint registered with the provider.
	HookURL    string
	Repository GitRepositoryRef
	// Webhooks is nil when no subscriptions are declared.
	Webhooks []GitRepositoryWebhook
}

// GitRepositoryRef identifies the repository on the provider.
type GitRepositoryRef struct {
	Owner string
	Name  string
}

// GitRepositoryWebhook is a single webhook subscription.
// Events contains the normalised provider event strings ("push", "pull_request" etc.).
type GitRepositoryWebhook struct {
	Events []string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveGitRepository decodes and validates the raw JSON contract from the
// GitRepository CR spec into a ResolvedGitRepository. Returns an error if the
// CR is nil, the contract is absent, or any required field is missing or
// malformed.
func ResolveGitRepository(repo *sourcesv1alpha1.GitRepository) (*ResolvedGitRepository, error) {
	if repo == nil {
		return nil, fmt.Errorf("gitrepository is nil")
	}

	if len(repo.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(repo.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode contract: %w", err)
	}

	// ------------------------------------------------
	// Required scalar fields.
	// ------------------------------------------------
	provider, err := mustString(raw, "provider")
	if err != nil {
		return nil, err
	}

	hookURL, err := mustString(raw, "hookUrl")
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// Repository ref (REQUIRED object).
	// ------------------------------------------------
	repoRaw, ok := raw["repository"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("repository is required")
	}

	owner, err := mustString(repoRaw, "owner")
	if err != nil {
		return nil, fmt.Errorf("repository.owner: %w", err)
	}

	name, err := mustString(repoRaw, "name")
	if err != nil {
		return nil, fmt.Errorf("repository.name: %w", err)
	}

	ref := GitRepositoryRef{Owner: owner, Name: name}

	// ------------------------------------------------
	// Webhooks (OPTIONAL).
	//
	// Entries with no valid event strings are skipped — an empty events
	// list produces no subscription and is not an error.
	// ------------------------------------------------
	var webhooks []GitRepositoryWebhook
	if hooksRaw, ok := raw["webhooks"].([]any); ok {
		for i, h := range hooksRaw {
			hookMap, ok := h.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("webhooks[%d] must be an object", i)
			}

			eventsRaw, ok := hookMap["events"].([]any)
			if !ok || len(eventsRaw) == 0 {
				continue
			}

			var events []string
			for _, e := range eventsRaw {
				if s, ok := e.(string); ok && s != "" {
					events = append(events, s)
				}
			}

			if len(events) > 0 {
				webhooks = append(webhooks, GitRepositoryWebhook{Events: events})
			}
		}
	}

	return &ResolvedGitRepository{
		Repository: repo,
		Spec: &ResolvedGitRepositorySpec{
			Provider:   provider,
			HookURL:    hookURL,
			Repository: ref,
			Webhooks:   webhooks,
		},
	}, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
// -----------------------------------------------------------------------------

// mustString extracts a non-empty string from m[key].
// Returns an error — resolution must never panic and crash the controller.
func mustString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("field %q must be a non-empty string", key)
	}
	return s, nil
}
