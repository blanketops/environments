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
This file owns the Provider interface for the routes domain — the contract
that all route backend implementations must satisfy.

A Provider is responsible for materializing or removing the runtime resource
(DomainMapping, Ingress, HTTPRoute) that exposes a workload at the declared
host and path. Ensure is idempotent — calling it multiple times for the same
Route CR must produce the same result with no unintended side effects.

The concrete implementation for the Knative runtime lives in knative.go.
Additional runtimes (Kubernetes Ingress, Gateway API HTTPRoute) are registered
alongside it as new files in this package.

Provider is pure in the sense that it never writes to the Route CR status —
that responsibility belongs to pkg/routes/application/status.go.

See also:
  - pkg/routes/api/knative.go         — Knative DomainMapping implementation
  - pkg/routes/application/backend_selector.go — selects Provider by Runtime
  - pkg/routes/application/service.go — calls Provider.Ensure and handles result
*/
package api

import (
	"context"

	"github.com/blanketops/environments/pkg/apis/route/domain"
	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

// Provider materializes and maintains the runtime resource for a Route CR.
// Implementations must be idempotent and must not mutate the Route CR directly.
type Provider interface {
	// Ensure creates or reconciles the runtime resource (DomainMapping, Ingress,
	// or HTTPRoute) for the given Route. Returns PhaseReady on success and
	// PhaseFailed on terminal failure. Non-terminal errors (e.g. transient API
	// failures) are returned as Go errors for the controller to requeue.
	Ensure(ctx context.Context, resolved *routeResolution.ResolvedRoute, route domain.Route) (domain.RouteResult, error)
}
