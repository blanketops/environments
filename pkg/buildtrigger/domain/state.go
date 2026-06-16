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
This file owns the BuildTrigger state machine types — State, Reason,
Transition, and BuildTriggerState.

State transitions are policy-driven, not imperative. The state machine records
what happened and why; it does not drive execution. Execution is handled by
the Provider and application service layers.
*/
package domain

import "time"

// State is the lifecycle state of a BuildTrigger.
type State string

const (
	// StatePending — trigger created, not yet evaluated.
	StatePending State = "Pending"
	// StateAccepted — trigger passed policy evaluation.
	StateAccepted State = "Accepted"
	// StateRejected — trigger failed policy evaluation.
	StateRejected State = "Rejected"
	// StateDelivered — trigger was successfully dispatched to its target.
	StateDelivered State = "Delivered"
	// StateFailed — dispatch failed due to infrastructure or orchestration error.
	StateFailed State = "Failed"
)

// Reason provides a structured, machine-readable explanation for a state
// transition. Used for condition Reason fields and audit records.
type Reason string

const (
	ReasonPolicyMismatch   Reason = "PolicyMismatch"
	ReasonDuplicateEvent   Reason = "DuplicateEvent"
	ReasonInvalidTarget    Reason = "InvalidTarget"
	ReasonExecutionStarted Reason = "ExecutionStarted"
	ReasonExecutionFailed  Reason = "ExecutionFailed"
	ReasonSystemError      Reason = "SystemError"
)

// Transition records a single state change with its reason and timestamp.
type Transition struct {
	From    State
	To      State
	Reason  Reason
	At      time.Time
	Message string
}

// BuildTriggerState is a snapshot of the full state machine for a BuildTrigger.
// Transitions is the ordered history of state changes. Retry tracking is
// policy-driven — MaxAttempts comes from the resolved trigger policy, not from
// imperative retry logic.
type BuildTriggerState struct {
	Current     State
	Transitions []Transition
	// Attempts is the number of execution attempts made so far.
	Attempts int
	// MaxAttempts is the policy limit. Zero means no retry.
	MaxAttempts int
}
