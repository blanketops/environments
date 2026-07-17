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
This file owns the Phase type and its constants for the ServiceUnit domain.

Phase mirrors blanketops.common.v1.ServiceUnitDeploymentPhase in the proto
contract. The status writer (pkg/apis/serviceunit/application/status.go) maps
Phase values to the conditions on the ServiceUnit CR status subresource.

This is a distinct concept from pkg/apis/deployment/domain/state.go's
ServiceUnitState, despite the name proximity: that type tracks a given
Deployment's own per-unit rollout progress once materializing a ServiceUnit
onto a chosen runtime. Phase here tracks the ServiceUnit CR's own lifecycle —
whether its image is resolved and ready to be materialized at all — which is
this CR's business regardless of whether any Deployment currently references
it.

See also:
  - pkg/apis/serviceunit/domain/result.go — Result carries Phase as its outcome
  - pkg/apis/serviceunit/application/status.go — writes Phase to the CR status
*/
package domain

// Phase represents the current lifecycle phase of a ServiceUnit CR.
// Set by the application service after each Reconcile call.
type Phase string

const (
	// PhasePending indicates the controller has accepted the ServiceUnit CR
	// but its image is not yet resolved (e.g. a BUILD unit awaiting its
	// source Build to produce an image).
	PhasePending Phase = "Pending"

	// PhaseDeploying indicates the image is resolved and the unit is
	// currently being materialized by a referencing Deployment.
	PhaseDeploying Phase = "Deploying"

	// PhaseReady indicates the unit's image is resolved and available.
	PhaseReady Phase = "Ready"

	// PhaseFailed indicates the unit could not be resolved — check
	// Result.Message and the CR conditions for detail.
	PhaseFailed Phase = "Failed"
)
