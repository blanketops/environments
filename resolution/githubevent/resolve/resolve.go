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
Package resolve implements resolution for the GitHubEvent CR.

The GitHubEvent CR stores its spec as a raw JSON contract (spec.contract).
ResolveGitHubEvent decodes this into a fully typed ResolvedGitHubEvent —
the authoritative runtime representation consumed by all downstream domain
and application logic.

GitHubEvent is an immutable record of an observed provider event. The spec
carries facts, not intent — fields like Repository, EventType, and CommitSHA
are copied verbatim from the webhook payload.

All failures surface as errors. Resolution never panics.
*/
package resolve

import (
	"encoding/json"
	"fmt"

	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedGitHubEvent pairs the original Kubernetes GitHubEvent object with
// its fully decoded and validated spec.
type ResolvedGitHubEvent struct {
	Event *eventsv1alpha1.GitHubEvent
	Spec  *ResolvedGitHubEventSpec
}

// ResolvedGitHubEventSpec is the decoded GitHubEvent spec.
// Fields are facts copied from the webhook payload — none are derived or
// transformed at this layer.
type ResolvedGitHubEventSpec struct {
	Repository string
	EventType  string
	// EventID is the provider-assigned delivery GUID. Used for idempotency
	// and audit. Temporarily optional — uncomment requireString call below
	// once all GitHubEvent CRs carry an eventId field.
	EventID   string
	Ref       string
	CommitSHA string
	Actor     string
	// Webhook carries the HMAC secret reference used for signature verification.
	// Optional — absent when the GitHubEvent is created via manual dispatch.
	Webhook ResolvedWebhook
}

// ResolvedWebhook is the resolved webhook configuration.
type ResolvedWebhook struct {
	SecretRef ResolvedSecretRef
}

// ResolvedSecretRef is the resolved Kubernetes Secret reference for the
// webhook HMAC secret.
type ResolvedSecretRef struct {
	Name string
	Key  string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveGitHubEvent decodes and validates the raw JSON contract from the
// GitHubEvent CR spec into a ResolvedGitHubEvent. Returns an error if the CR
// is nil, the contract is absent, or any required field is missing or malformed.
func ResolveGitHubEvent(ev *eventsv1alpha1.GitHubEvent) (*ResolvedGitHubEvent, error) {
	if ev == nil {
		return nil, fmt.Errorf("GitHubEvent is nil")
	}

	if len(ev.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(ev.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Required fields — provider event facts.
	// ------------------------------------------------
	repo, err := requireString(raw, "repository")
	if err != nil {
		return nil, err
	}

	eventType, err := requireString(raw, "eventType")
	if err != nil {
		return nil, err
	}

	// TODO: promote eventId to required once all CRs carry the field.
	// eventID, err := requireString(raw, "eventId")
	// if err != nil {
	// 	return nil, err
	// }

	// ------------------------------------------------
	// Webhook (OPTIONAL).
	//
	// Absent for manual dispatch — resolveWebhook returns a zero
	// ResolvedWebhook when the field is not present.
	// ------------------------------------------------
	webhook, err := resolveWebhook(raw)
	if err != nil {
		return nil, err
	}

	spec := &ResolvedGitHubEventSpec{
		Repository: repo,
		EventType:  eventType,
		EventID:    optionalString(raw, "eventId"),
		Ref:        optionalString(raw, "ref"),
		CommitSHA:  optionalString(raw, "commitSHA"), // FIXED: was "commitSha"
		Actor:      optionalString(raw, "actor"),
		Webhook:    webhook,
	}

	return &ResolvedGitHubEvent{
		Event: ev,
		Spec:  spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Webhook resolution
// -----------------------------------------------------------------------------

// resolveWebhook extracts and validates the optional webhook.secretRef block.
// Returns a zero ResolvedWebhook when the field is absent — callers must check
// whether SecretRef.Name is non-empty before using it.
func resolveWebhook(raw map[string]any) (ResolvedWebhook, error) {
	var w ResolvedWebhook

	webhookRaw, ok := raw["webhook"]
	if !ok {
		return w, nil
	}

	webhookMap, ok := webhookRaw.(map[string]any)
	if !ok {
		return w, fmt.Errorf("field \"webhook\" must be an object")
	}

	secretRaw, ok := webhookMap["secretRef"]
	if !ok {
		return w, nil
	}

	secretMap, ok := secretRaw.(map[string]any)
	if !ok {
		return w, fmt.Errorf("field \"webhook.secretRef\" must be an object")
	}

	name, err := requireString(secretMap, "name")
	if err != nil {
		return w, fmt.Errorf("webhook.secretRef.name: %w", err)
	}

	key, err := requireString(secretMap, "key")
	if err != nil {
		return w, fmt.Errorf("webhook.secretRef.key: %w", err)
	}

	w.SecretRef = ResolvedSecretRef{Name: name, Key: key}
	return w, nil
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
