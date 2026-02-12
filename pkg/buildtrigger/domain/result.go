package domain

import "time"

// Result represents the outcome produced by a BuildTrigger.
type Result struct {
	// Whether the trigger resulted in an execution being started
	Triggered bool

	// Human-readable summary of the outcome
	Message string

	// Reference to the execution that was triggered
	// (e.g. Build name, BuildRun name, or external ref)
	ExecutionRef string

	// ExecutionKind identifies what was triggered
	// Examples: "Build", "BuildRun", "ExternalPipeline"
	ExecutionKind string

	// CorrelationID ties this trigger result back to an external event
	// (e.g. GitHub delivery ID, webhook UUID)
	CorrelationID string

	// When the result was produced
	At time.Time
}
