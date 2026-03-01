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
