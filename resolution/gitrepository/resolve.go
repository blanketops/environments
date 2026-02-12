package gitrepository

import (
	"encoding/json"
	"fmt"

	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
)

//
// ==============================
// RUNTIME GIT REPOSITORY (AUTHORITATIVE)
// ==============================
//

// ResolvedGitRepository is the SINGLE runtime representation
// of a GitRepository contract.
type ResolvedGitRepository struct {
	Repository *sourcesv1alpha1.GitRepository
	Spec       *ResolvedGitRepositorySpec
}

type ResolvedGitRepositorySpec struct {
	Provider   string
	HookURL    string
	Repository GitRepositoryRef
	Webhooks   []GitRepositoryWebhook
}

type GitRepositoryRef struct {
	Owner string
	Name  string
}

type GitRepositoryWebhook struct {
	Events []string
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolveGitRepository(
	repo *sourcesv1alpha1.GitRepository,
) (*ResolvedGitRepository, error) {

	if repo == nil {
		return nil, fmt.Errorf("gitrepository is nil")
	}

	if len(repo.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// ------------------------------------------------
	// Decode raw contract
	// ------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(repo.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode contract: %w", err)
	}

	// ------------------------------------------------
	// Resolve provider (MANDATORY)
	// ------------------------------------------------
	provider := mustString(raw, "provider")

	// ------------------------------------------------
	// Resolve hook URL (MANDATORY)
	// ------------------------------------------------
	hookURL := mustString(raw, "hookUrl")

	// ------------------------------------------------
	// Resolve repository ref (MANDATORY)
	// ------------------------------------------------
	repoRaw, ok := raw["repository"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("repository is required")
	}

	ref := GitRepositoryRef{
		Owner: mustString(repoRaw, "owner"),
		Name:  mustString(repoRaw, "name"),
	}

	// ------------------------------------------------
	// Resolve webhooks (OPTIONAL)
	// ------------------------------------------------
	var webhooks []GitRepositoryWebhook

	if hooksRaw, ok := raw["webhooks"].([]any); ok {
		for _, h := range hooksRaw {
			hookMap, ok := h.(map[string]any)
			if !ok {
				continue
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
				webhooks = append(webhooks, GitRepositoryWebhook{
					Events: events,
				})
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

//
// ==============================
// HELPERS (MATCH BUILD STYLE)
// ==============================
//

func mustString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		panic(fmt.Sprintf("missing required field %q", key))
	}
	s, ok := v.(string)
	if !ok || s == "" {
		panic(fmt.Sprintf("field %q must be a non-empty string", key))
	}
	return s
}
