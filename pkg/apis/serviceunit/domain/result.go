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
This file owns Result — the outcome value produced by ServiceUnitService.Reconcile.
The application service (pkg/apis/serviceunit/application/service.go) derives
this directly from the resolved spec and passes it to the status writer to
update the ServiceUnit CR.

Result is a plain value type — no pointers, no interfaces. Callers check
Ready() to determine whether the unit's image is resolved before treating it
as consumable by a Deployment.

See also:
  - pkg/apis/serviceunit/domain/state.go     — Phase constants carried in Result
  - pkg/apis/serviceunit/application/status.go — consumes Result to write CR status
*/
package domain

// Result is the outcome of a single ServiceUnitService.Reconcile call.
type Result struct {
	// Phase is the resulting lifecycle phase after resolution.
	Phase Phase

	// Message carries human-readable detail on the outcome.
	// Always populated on Pending/Failed; may be empty on Ready.
	Message string
}

// Ready returns true when the unit's image is resolved and available
// (Phase == PhaseReady). Callers use this as the predicate before treating
// the unit as consumable by a Deployment.
func (r Result) Ready() bool {
	return r.Phase == PhaseReady
}
