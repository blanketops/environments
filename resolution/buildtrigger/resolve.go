package buildtrigger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

//
// ==============================
// RUNTIME BUILD TRIGGER (AUTHORITATIVE)
// ==============================
//

type ResolvedBuildTrigger struct {
	Trigger *environmentv1.BuildTrigger
	Spec    *ResolvedBuildTriggerSpec
}

type ResolvedBuildTriggerSpec struct {
	// Identity
	TriggerID string
	EventID   string

	// Normalized enums
	Source domain.TriggerSource
	Type   domain.TriggerType

	// Repository context
	Repository string
	Ref        string

	// Optional metadata
	SHA   string
	Actor string

	// Audit
	PayloadHash string

	// Time (authoritative)
	OccurredAt time.Time
	ReceivedAt time.Time

	// Target
	Target struct {
		Name      string
		Namespace string
	}

	// Policy
	PayloadPolicy *ResolvedPayloadPolicy
}

type ResolvedPayloadPolicy struct {
	Allow bool
}

//
// ==============================
// RESOLUTION ENTRY POINT
// ==============================
//

func ResolveBuildTrigger(
	trigger *environmentv1.BuildTrigger,
) (*ResolvedBuildTrigger, error) {

	if trigger == nil {
		return nil, fmt.Errorf("buildtrigger is nil")
	}
	if len(trigger.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	now := time.Now().UTC()

	// -----------------------------------------------------------------
	// Decode raw contract
	// -----------------------------------------------------------------
	var raw map[string]any
	if err := json.Unmarshal(trigger.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// -----------------------------------------------------------------
	// Resolve mandatory fields
	// -----------------------------------------------------------------
	source := domain.TriggerSource(mustString(raw, "source"))
	eventType := domain.TriggerType(mustString(raw, "eventType"))
	ref := mustString(raw, "ref")
	eventID := mustString(raw, "eventId")

	// -----------------------------------------------------------------
	// Resolve repository
	// -----------------------------------------------------------------
	repoRaw := mustObject(raw, "repository")
	repository := fmt.Sprintf(
		"%s/%s",
		mustString(repoRaw, "owner"),
		mustString(repoRaw, "name"),
	)

	// -----------------------------------------------------------------
	// Resolve buildRef
	// -----------------------------------------------------------------
	buildRaw := mustObject(raw, "buildRef")
	buildRef := mustString(buildRaw, "name")

	// -----------------------------------------------------------------
	// Optional fields
	// -----------------------------------------------------------------
	sha := optionalString(raw, "sha")
	actor := optionalString(raw, "actor")

	// -----------------------------------------------------------------
	// Deterministic identity
	// -----------------------------------------------------------------
	triggerID := computeTriggerID(
		source,
		eventID,
		eventType,
		buildRef,
		trigger.Namespace,
	)

	spec := &ResolvedBuildTriggerSpec{
		TriggerID:  triggerID,
		EventID:    eventID,
		Source:     source,
		Type:       eventType,
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

	// Optional policy
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

//
// ==============================
// HELPERS
// ==============================
//

func computeTriggerID(
	source domain.TriggerSource,
	eventID string,
	eventType domain.TriggerType,
	buildRef string,
	namespace string,
) string {

	key := fmt.Sprintf(
		"source=%s|event=%s|type=%s|target=Build/%s/%s",
		source,
		eventID,
		eventType,
		namespace,
		buildRef,
	)

	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func mustObject(m map[string]any, key string) map[string]any {
	v, ok := m[key].(map[string]any)
	if !ok {
		panic(fmt.Sprintf("field %q must be an object", key))
	}
	return v
}

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
