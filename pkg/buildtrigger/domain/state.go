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

// State represents the lifecycle state of a BuildTrigger.
type State string

const (
	// Trigger has been created but not yet evaluated
	StatePending State = "Pending"

	// Trigger was evaluated and accepted by policy
	StateAccepted State = "Accepted"

	// Trigger was evaluated but rejected by policy
	StateRejected State = "Rejected"

	// Trigger has been successfully delivered to its target
	// (e.g. Build execution was triggered)
	StateDelivered State = "Delivered"

	// Trigger delivery failed (infrastructure or orchestration failure)
	StateFailed State = "Failed"
)

// Reason provides structured explanation for state transitions.
type Reason string

const (
	ReasonPolicyMismatch   Reason = "PolicyMismatch"
	ReasonDuplicateEvent   Reason = "DuplicateEvent"
	ReasonInvalidTarget    Reason = "InvalidTarget"
	ReasonExecutionStarted Reason = "ExecutionStarted"
	ReasonExecutionFailed  Reason = "ExecutionFailed"
	ReasonSystemError      Reason = "SystemError"
)

// Transition records a single state change.
type Transition struct {
	From    State
	To      State
	Reason  Reason
	At      time.Time
	Message string
}

// BuildTriggerState is the full state machine snapshot.
type BuildTriggerState struct {
	Current     State
	Transitions []Transition

	// Retry tracking (policy-driven, not imperative)
	Attempts    int
	MaxAttempts int
}
