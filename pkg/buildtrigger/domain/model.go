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

package domain

import "time"

//
// ─────────────────────────────────────────────────────────────
// Strong types (DO NOT use raw strings in domain logic)
// ─────────────────────────────────────────────────────────────
//

// TriggerSource identifies the external system that emitted the event.
type TriggerSource string

const (
	TriggerSourceGitHub TriggerSource = "github"
	TriggerSourceGitLab TriggerSource = "gitlab"
	TriggerSourceManual TriggerSource = "manual"
)

// TriggerType describes what kind of event occurred.
type TriggerType string

const (
	TriggerTypePush        TriggerType = "push"
	TriggerTypePullRequest TriggerType = "pull_request"
	TriggerTypeManual      TriggerType = "manual"
	TriggerTypeSchedule    TriggerType = "schedule"
)

// TargetKind limits what a trigger may target.
// DO NOT allow arbitrary kinds.
type TargetKind string

const (
	TargetKindBuild TargetKind = "Build"
)

//
// ─────────────────────────────────────────────────────────────
// Core domain objects
// ─────────────────────────────────────────────────────────────
//

// Trigger represents a normalized, immutable trigger event.
//
// IMMUTABILITY RULE:
// - Trigger is immutable once created.
// - Any semantic change MUST produce a new Trigger with a new ID.
type Trigger struct {
	// Deterministic internal ID (hash of Source + EventID + Target)
	ID string

	// External system that emitted the event
	Source TriggerSource

	// Type of event
	Type TriggerType

	// Repository in owner/name form
	Repository string

	// Git ref associated with the event
	Ref string

	// Commit SHA (if known)
	SHA string

	// Actor that caused the event
	Actor string

	// External event identifier (e.g. GitHub delivery ID)
	EventID string

	// Optional payload checksum/signature (audit/debug)
	PayloadHash string

	// When the event occurred at the source
	OccurredAt time.Time

	// When the system accepted the trigger
	ReceivedAt time.Time
}

// Target describes what this trigger is meant to affect.
// IMPORTANT: This does NOT execute anything.
type Target struct {
	Kind      TargetKind
	Name      string
	Namespace string
}

// BuildTrigger is the aggregate root for trigger handling.
// It owns validation, deduplication, and acceptance decisions.
type BuildTrigger struct {
	Trigger Trigger
	Target  Target
}

// BuildTriggerStatus represents the outcome of evaluating a BuildTrigger.
//
// This is a PURE domain object.
// It is serialized into CR status.contract by the application layer.
type BuildTriggerStatus struct {
	// Was the trigger accepted by policy?
	Accepted bool

	// Did the trigger result in an execution?
	Triggered bool

	// Human-readable explanation (policy or execution message)
	Message string

	// Reference to the triggered execution (e.g. BuildRun name)
	TriggeredRef string
}
