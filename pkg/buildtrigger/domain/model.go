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
This file owns the core domain types for the BuildTrigger domain — strong
source and type enums, the immutable Trigger aggregate, Target, BuildTrigger,
and BuildTriggerStatus.

Strong types are used throughout — raw strings are not permitted in domain
logic. The Mapper (pkg/buildtrigger/application/mapper.go) is the only place
that constructs these from strings.

Trigger is immutable once created. Any semantic change must produce a new
Trigger with a new ID — ID stability is the deduplication guarantee.
*/
package domain

import "time"

// TriggerSource identifies the external system that emitted the event.
type TriggerSource string

const (
	TriggerSourceGitHub TriggerSource = "github"
	TriggerSourceGitLab TriggerSource = "gitlab"
	TriggerSourceManual TriggerSource = "manual"
)

// TriggerType describes the kind of event that occurred.
type TriggerType string

const (
	TriggerTypePush        TriggerType = "commit"
	TriggerTypePullRequest TriggerType = "pull_request"
	TriggerTypeManual      TriggerType = "manual"
	TriggerTypeSchedule    TriggerType = "schedule"
)

// TargetKind constrains what a trigger may target.
// Only Build is currently supported — extend here when new target kinds are added.
type TargetKind string

const (
	TargetKindBuild TargetKind = "Build"
)

// Trigger is a normalised, immutable record of a single trigger event.
// ID is a deterministic hash of Source + EventID + Target — equal inputs
// always produce the same ID, enabling deduplication without external state.
type Trigger struct {
	// ID is the deterministic deduplication key.
	ID string

	Source TriggerSource
	Type   TriggerType

	// Repository is the "owner/name" identifier of the source repository.
	Repository string
	// Ref is the Git ref the event applies to.
	Ref string
	// SHA is the commit SHA if known.
	SHA string
	// Actor is the provider login of the user who caused the event.
	Actor string
	// EventID is the provider-assigned delivery ID (e.g. GitHub delivery GUID).
	EventID string
	// PayloadHash is a checksum of the raw webhook payload for audit use.
	PayloadHash string

	// OccurredAt is when the event occurred at the provider.
	OccurredAt time.Time
	// ReceivedAt is when the platform accepted the trigger.
	ReceivedAt time.Time
}

// Target describes the CR this trigger will fire. It carries intent only —
// no execution happens at the domain layer.
type Target struct {
	Kind      TargetKind
	Name      string
	Namespace string
}

// BuildTrigger is the aggregate root for trigger handling. It owns validation,
// deduplication, and acceptance decisions via the Provider interface.
type BuildTrigger struct {
	Trigger Trigger
	Target  Target
}

// BuildTriggerStatus is the pure domain outcome of evaluating a BuildTrigger.
// Serialised into the BuildTrigger CR's status.contract field by the
// StatusWriter (pkg/buildtrigger/application/status.go).
type BuildTriggerStatus struct {
	// Accepted indicates the trigger passed policy evaluation.
	Accepted bool
	// Triggered indicates the trigger resulted in an execution being dispatched.
	Triggered bool
	// Message is a human-readable explanation of the outcome.
	Message string
	// TriggeredRef is the name of the execution that was dispatched.
	TriggeredRef string
}
