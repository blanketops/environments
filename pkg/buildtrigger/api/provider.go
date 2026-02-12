package api

import (
	"context"

	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/buildtrigger"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/domain"
)

// Provider is the contract between the application layer
// and the outside world (event sources, policies, rule engines).
//
// The application NEVER hard-codes policy.
// It asks the provider to evaluate intent.
type Provider interface {

	// Evaluate determines whether a BuildTrigger
	// should be accepted, ignored, or rejected.
	//
	// PARAMETERS:
	// - resolved → authoritative runtime facts (CR, timestamps, IDs)
	// - trigger  → pure domain projection
	//
	// GUARANTEES:
	// - Pure decision (no side effects)
	// - Deterministic for same inputs
	// - Idempotent
	//
	// DOES NOT:
	// - Create BuildRuns
	// - Mutate Kubernetes
	// - Talk to external systems
	Evaluate(
		ctx context.Context,
		resolved *buildtriggerResolution.ResolvedBuildTrigger,
		trigger domain.BuildTrigger,
	) (domain.Decision, error)
}
