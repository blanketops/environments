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

package githubevent

import (
	"encoding/json"
	"fmt"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
)

//
// ==============================
// RUNTIME GitHubEvent (AUTHORITATIVE)
// ==============================
//

// ResolvedGitHubEvent is the SINGLE runtime representation.
// Everything downstream MUST use this.
type ResolvedGitHubEvent struct {
	Event *eventsv1alpha1.GitHubEvent
	Spec  *ResolvedGitHubEventSpec
}

type ResolvedGitHubEventSpec struct {
	// Core event facts
	Repository string
	EventType  string
	EventID    string

	Ref       string
	CommitSHA string
	Actor     string

	// Webhook (resolved from contract)
	Webhook ResolvedWebhook
}

type ResolvedWebhook struct {
	SecretRef ResolvedSecretRef
}

type ResolvedSecretRef struct {
	Name string
	Key  string
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolveGitHubEvent(ev *eventsv1alpha1.GitHubEvent) (*ResolvedGitHubEvent, error) {
	if ev == nil {
		return nil, fmt.Errorf("GitHubEvent is nil")
	}
	if len(ev.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// -----------------------------------------------------------------
	// Decode raw contract (EXACTLY like Build)
	// -----------------------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(ev.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// -----------------------------------------------------------------
	// REQUIRED FIELDS (FACTS, NOT INTENT)
	// -----------------------------------------------------------------
	repo, err := requireString(raw, "repository")
	if err != nil {
		return nil, err
	}
	eventType, err := requireString(raw, "eventType")
	if err != nil {
		return nil, err
	}
	// eventID, err := requireString(raw, "eventId")
	// if err != nil {
	// 	return nil, err
	// }

	webhook, err := resolveWebhook(raw)
	if err != nil {
		return nil, err
	}

	spec := &ResolvedGitHubEventSpec{
		Repository: repo,
		EventType:  eventType,
		//EventID:    eventID,

		Ref:       optionalString(raw, "ref"),
		CommitSHA: optionalString(raw, "commitSha"),
		Actor:     optionalString(raw, "actor"),

		Webhook: webhook,
	}

	return &ResolvedGitHubEvent{
		Event: ev,
		Spec:  spec,
	}, nil
}

//
// ==============================
// WEBHOOK RESOLUTION
// ==============================
//

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
		return w, err
	}
	key, err := requireString(secretMap, "key")
	if err != nil {
		return w, err
	}

	w.SecretRef = ResolvedSecretRef{
		Name: name,
		Key:  key,
	}

	return w, nil
}

//
// ==============================
// HELPERS (STRICT, NOT PANIC)
// ==============================
//

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
