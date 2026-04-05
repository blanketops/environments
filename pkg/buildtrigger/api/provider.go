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

package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
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
