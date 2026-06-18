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
Package githubpayload implements resolution for the GitHubPayload CR.

The GitHubPayload CR stores its spec as a raw JSON contract (spec.contract),
identical in shape to GitHubEvent. ResolveGitHubPayload decodes this into a
fully typed ResolvedGitHubPayload — the authoritative runtime representation
consumed by BuildTrigger's contract-matching logic.

GitHubPayload is an immutable, platform-generated record of a single webhook
delivery. Fields are facts copied verbatim from the Sensor's trigger
parameters — none are derived or transformed at this layer.

All failures surface as errors. Resolution never panics.
*/
package githubpayload

import (
	"encoding/json"
	"fmt"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedGitHubPayload pairs the original Kubernetes GitHubPayload object
// with its fully decoded and validated spec.
type ResolvedGitHubPayload struct {
	Payload *eventsv1alpha1.GitHubPayload
	Spec    *ResolvedGitHubPayloadSpec
}

// ResolvedGitHubPayloadSpec is the decoded GitHubPayload spec.
// Fields are facts copied from the Sensor's trigger parameters — none are
// derived or transformed at this layer.
type ResolvedGitHubPayloadSpec struct {
	Repository string
	EventType  string
	EventID    string
	Ref        string
	CommitSHA  string
	Actor      string
	// GitHubEventRef is the name of the user-facing GitHubEvent CR this
	// payload corresponds to — the traceability link BuildTrigger records
	// in its own status on match.
	GitHubEventRef string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveGitHubPayload decodes and validates the raw JSON contract from the
// GitHubPayload CR spec into a ResolvedGitHubPayload. Returns an error if the
// CR is nil, the contract is absent, or any required field is missing or
// malformed.
func ResolveGitHubPayload(p *eventsv1alpha1.GitHubPayload) (*ResolvedGitHubPayload, error) {
	if p == nil {
		return nil, fmt.Errorf("GitHubPayload is nil")
	}
	if len(p.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(p.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	repo, err := requireString(raw, "repository")
	if err != nil {
		return nil, err
	}
	eventType, err := requireString(raw, "eventType")
	if err != nil {
		return nil, err
	}
	githubEventRef, err := requireString(raw, "githubEventRef")
	if err != nil {
		return nil, err
	}

	spec := &ResolvedGitHubPayloadSpec{
		Repository:     repo,
		EventType:      eventType,
		EventID:        optionalString(raw, "eventId"),
		Ref:            optionalString(raw, "ref"),
		CommitSHA:      optionalString(raw, "commitSHA"),
		Actor:          optionalString(raw, "actor"),
		GitHubEventRef: githubEventRef,
	}

	return &ResolvedGitHubPayload{
		Payload: p,
		Spec:    spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
// -----------------------------------------------------------------------------

// requireString extracts a non-empty string from m[key].
// Returns an error — resolution must never panic and crash the controller.
func requireString(m map[string]any, key string) (string, error) {
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

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
