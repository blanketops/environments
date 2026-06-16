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
This file owns the Provider interface for the BuildTrigger domain — the
contract between the application layer and trigger evaluation backends.

A Provider answers one question: should this trigger be accepted, ignored,
or rejected? It is a pure decision function — no side effects, no Kubernetes
mutations, no external calls. All execution (BuildRun creation) is handled
upstream by the application service after the Provider returns Accept.

Concrete implementations include the GitHub provider
(pkg/buildtrigger/api/github.go) which evaluates branch policy, event type
allowlists, and duplicate event detection.
*/
package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
)

// Provider evaluates a resolved BuildTrigger and returns a Decision.
// Implementations must be pure — same inputs always produce the same Decision,
// with no side effects and no Kubernetes mutations.
type Provider interface {
	// Evaluate determines whether a BuildTrigger should be accepted, ignored,
	// or rejected based on policy and the resolved trigger facts.
	//
	//   - resolved → authoritative runtime facts (CR, timestamps, event IDs)
	//   - trigger  → pure domain projection for policy evaluation
	//
	// Returns a Decision and a non-nil error only when evaluation itself fails
	// (e.g. policy backend unreachable). A rejected trigger is not an error —
	// it is a valid Decision with Outcome=Reject.
	Evaluate(
		ctx context.Context,
		resolved *buildtriggerResolution.ResolvedBuildTrigger,
		trigger domain.BuildTrigger,
	) (domain.Decision, error)
}
