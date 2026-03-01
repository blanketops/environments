package domain

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// -----------------------------------------------------------------------------
// BuildSpec
// -----------------------------------------------------------------------------
//
// Pure internal specification of a Build.
// This is NOT the CRD shape.
// This is the canonical semantic input to execution.
type BuildSpec struct {
	// --------------------
	// Source
	// --------------------
	SourceURL   string
	ContextDir  string
	Revision    string
	CloneSecret string

	// --------------------
	// Strategy
	// --------------------
	StrategyName string
	StrategyKind string

	// --------------------
	// Output
	// --------------------
	Image string

	// --------------------
	// Execution identity
	// --------------------
	ServiceAccountName   string
	ServiceAccountSecret string

	// --------------------
	// Execution controls
	// --------------------
	TimeoutSeconds *int64 // nil = provider default

	// --------------------
	// Parameters
	// --------------------
	Params map[string]string

	// --------------------
	// Policy (NEW)
	// --------------------
	Policy *BuildPolicy
}

// -----------------------------------------------------------------------------
// BuildPolicy
// -----------------------------------------------------------------------------
//
// Pure domain-level execution policy.
// Fully resolved, immutable at runtime.
type BuildPolicy struct {
	Retry *RetryPolicy
}

// -----------------------------------------------------------------------------
// RetryPolicy
// -----------------------------------------------------------------------------
//
// Defines retry semantics for failed executions.
type RetryPolicy struct {
	// Retry only when execution fails
	OnFailure bool

	// Maximum total execution attempts (including first run)
	MaxAttempts int32
}

// -----------------------------------------------------------------------------
// BuildStatus
// -----------------------------------------------------------------------------
//
// Internal domain status for a Build execution.
// This is serialized into CRD status.contract.
type BuildStatus struct {
	// Was an execution triggered?
	Triggered bool

	// Did the execution succeed?
	Success bool

	// Human-readable message
	Message string

	// Reference to the execution object (e.g. BuildRun name)
	ExecutionRef string

	// Deterministic execution hash
	BuildHash string

	// --------------------
	// Retry (AUTHORITATIVE)
	// --------------------

	OnFailure bool

	// Last failure timestamp (optional, observability only)
	LastFailureAt *metav1.Time
}
