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
This file owns RouteResult — the outcome value returned by every Provider.Ensure
call. The application service (pkg/routes/application/service.go) receives this
and passes it to the status writer to update the Route CR.

RouteResult is a plain value type — no pointers, no interfaces. Callers check
Materialized() to determine whether the provider completed successfully before
inspecting ResolvedAddress.

See also:
  - pkg/routes/domain/state.go     — Phase constants carried in RouteResult
  - pkg/routes/api/provider.go     — Provider.Ensure returns RouteResult
  - pkg/routes/application/status.go — consumes RouteResult to write CR status
*/
package domain

// RouteResult is the outcome of a single Provider.Ensure call.
// Returned by every backend implementation (Knative, Kubernetes, Gateway API).
type RouteResult struct {
	// Phase is the resulting lifecycle phase after the Ensure call.
	Phase Phase

	// ResolvedAddress is the serving URL or load balancer hostname once the
	// route is materialized. Empty when Phase is not PhaseReady.
	ResolvedAddress string

	// Message carries human-readable detail on the outcome.
	// Always populated on failure; may be empty on success.
	Message string
}

// Materialized returns true when the provider successfully provisioned the
// route (Phase == PhaseReady). Callers use this as the success predicate
// before reading ResolvedAddress.
func (r RouteResult) Materialized() bool {
	return r.Phase == PhaseReady
}
