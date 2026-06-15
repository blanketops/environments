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
Package buildtrigger implements resolution for the BuildTrigger CR.

The BuildTrigger CR stores its spec as a raw JSON contract (spec.contract).
ResolveBuildTrigger decodes this into a fully typed ResolvedBuildTrigger —
the authoritative runtime representation consumed by all downstream domain
and application logic.

A deterministic TriggerID is computed from the source, event ID, event type,
build target, and namespace — enabling deduplication across webhook redeliveries
without consulting external state.

All failures surface as errors. Resolution never panics.
*/
package buildtrigger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	environmentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedBuildTrigger pairs the original Kubernetes BuildTrigger object with
// its fully decoded and validated spec.
type ResolvedBuildTrigger struct {
	Trigger *environmentv1alpha1.BuildTrigger
	Spec    *ResolvedBuildTriggerSpec
}

// ResolvedBuildTriggerSpec is the decoded and validated BuildTrigger spec.
type ResolvedBuildTriggerSpec struct {
	// TriggerID is a deterministic SHA-256 hash derived from source, eventID,
	// eventType, buildRef, and namespace. Used for deduplication.
	TriggerID string
	// EventID is the provider-assigned delivery GUID. Duplicate event IDs are
	// rejected by the domain layer.
	EventID string

	// Source and Type are normalised domain enums, not raw strings.
	Source domain.TriggerSource
	Type   domain.TriggerType

	// Repository is the fully qualified "owner/name" identifier.
	Repository string
	// Ref is the Git ref the event applies to (branch, tag, or PR head).
	Ref string

	// SHA and Actor are optional — populated when present in the payload.
	SHA   string
	Actor string

	// PayloadHash is a SHA-256 of the raw contract bytes for audit use.
	PayloadHash string

	// OccurredAt and ReceivedAt are both set to UTC now at resolution time.
	// OccurredAt should be overridden by the caller with the provider timestamp
	// when available.
	OccurredAt time.Time
	ReceivedAt time.Time

	// Target is the Build CR this trigger will fire.
	Target struct {
		Name      string
		Namespace string
	}

	// PayloadPolicy is optional — controls whether this trigger is admitted.
	PayloadPolicy *ResolvedPayloadPolicy
}

// ResolvedPayloadPolicy is the optional admission policy for the trigger.
type ResolvedPayloadPolicy struct {
	// Allow gates trigger admission — false suppresses BuildRun dispatch.
	Allow bool
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveBuildTrigger decodes and validates the raw JSON contract from the
// BuildTrigger CR spec into a ResolvedBuildTrigger. Returns an error if the
// CR is nil, the contract is absent, or any required field is missing or
// malformed.
func ResolveBuildTrigger(trigger *environmentv1alpha1.BuildTrigger) (*ResolvedBuildTrigger, error) {
	if trigger == nil {
		return nil, fmt.Errorf("buildtrigger is nil")
	}

	if len(trigger.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	now := time.Now().UTC()

	var raw map[string]any
	if err := json.Unmarshal(trigger.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Required scalar fields.
	// ------------------------------------------------
	sourceStr, err := mustString(raw, "source")
	if err != nil {
		return nil, err
	}

	eventTypeStr, err := mustString(raw, "eventType")
	if err != nil {
		return nil, err
	}

	ref, err := mustString(raw, "ref")
	if err != nil {
		return nil, err
	}

	eventID, err := mustString(raw, "eventId")
	if err != nil {
		return nil, err
	}

	// ------------------------------------------------
	// Repository (REQUIRED object).
	//
	// Projected as "owner/name" — matches the BuildTrigger proto contract
	// and the GitHub webhook payload shape.
	// ------------------------------------------------
	repoRaw, err := mustObject(raw, "repository")
	if err != nil {
		return nil, err
	}

	owner, err := mustString(repoRaw, "owner")
	if err != nil {
		return nil, fmt.Errorf("repository.owner: %w", err)
	}

	name, err := mustString(repoRaw, "name")
	if err != nil {
		return nil, fmt.Errorf("repository.name: %w", err)
	}

	repository := owner + "/" + name

	// ------------------------------------------------
	// BuildRef (REQUIRED object).
	//
	// Identifies the Build CR to patch when this trigger fires.
	// Resolved in the same namespace as the BuildTrigger.
	// ------------------------------------------------
	buildRaw, err := mustObject(raw, "buildRef")
	if err != nil {
		return nil, err
	}

	buildRef, err := mustString(buildRaw, "name")
	if err != nil {
		return nil, fmt.Errorf("buildRef.name: %w", err)
	}

	// ------------------------------------------------
	// Optional fields.
	// ------------------------------------------------
	sha := optionalString(raw, "sha")
	actor := optionalString(raw, "actor")

	// ------------------------------------------------
	// Deterministic TriggerID.
	//
	// Computed from the immutable identity fields so duplicate webhook
	// deliveries produce the same ID and can be idempotently rejected.
	// ------------------------------------------------
	triggerID := computeTriggerID(
		domain.TriggerSource(sourceStr),
		eventID,
		domain.TriggerType(eventTypeStr),
		buildRef,
		trigger.Namespace,
	)

	spec := &ResolvedBuildTriggerSpec{
		TriggerID:  triggerID,
		EventID:    eventID,
		Source:     domain.TriggerSource(sourceStr),
		Type:       domain.TriggerType(eventTypeStr),
		Repository: repository,
		Ref:        ref,
		SHA:        sha,
		Actor:      actor,
		OccurredAt: now,
		ReceivedAt: now,
		Target: struct {
			Name      string
			Namespace string
		}{
			Name:      buildRef,
			Namespace: trigger.Namespace,
		},
	}

	// ------------------------------------------------
	// PayloadPolicy (OPTIONAL).
	// ------------------------------------------------
	if ppRaw, ok := raw["payloadPolicy"].(map[string]any); ok {
		spec.PayloadPolicy = &ResolvedPayloadPolicy{
			Allow: optionalBool(ppRaw, "allow"),
		}
	}

	return &ResolvedBuildTrigger{
		Trigger: trigger,
		Spec:    spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// computeTriggerID returns a deterministic SHA-256 hex string derived from the
// trigger's immutable identity fields. Equal inputs always produce equal IDs,
// enabling deduplication without external state.
func computeTriggerID(
	source domain.TriggerSource,
	eventID string,
	eventType domain.TriggerType,
	buildRef string,
	namespace string,
) string {
	key := fmt.Sprintf(
		"source=%s|event=%s|type=%s|target=Build/%s/%s",
		source, eventID, eventType, namespace, buildRef,
	)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// mustObject extracts a nested object from m[key].
// Returns an error instead of panicking — resolution must never crash the
// controller process.
func mustObject(m map[string]any, key string) (map[string]any, error) {
	v, ok := m[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %q must be an object", key)
	}
	return v, nil
}

// mustString extracts a non-empty string from m[key].
// Returns an error instead of panicking — resolution must never crash the
// controller process.
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

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func optionalBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
