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
This file owns RouteService — the application service that orchestrates the
route reconciliation pipeline.

RouteService is the single entry point called by the route controller after
resolution. It sequences four steps in order:

 1. Map the resolved Route contract to a domain Route (Mapper).
 2. Select the correct provider for the runtime (BackendSelector).
 3. Dispatch via the provider (Provider.Ensure).
 4. Write the outcome back to the Route CR status (StatusWriter).

RouteService does not own retry logic, predicate evaluation, or status
persistence — those belong to the controller and writer layers respectively.

Condition derivation lives here: the service knows what Ready/Pending/Failed
mean for all registered runtimes. The Ready message is runtime-agnostic —
providers return a ResolvedAddress and the service constructs the condition.
The StatusWriter is a dumb persister.
*/
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/blanketops/environments/pkg/apis/route/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

// RouteService orchestrates the route reconciliation pipeline.
// It is stateless beyond its collaborators and safe for concurrent use.
type RouteService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

// NewRouteService constructs a RouteService with the required collaborators.
func NewRouteService(mapper *Mapper, status *StatusWriter, backend *BackendSelector) *RouteService {
	return &RouteService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

// Reconcile executes the full route pipeline for a resolved Route CR.
// It maps, selects, dispatches, and writes status in sequence. An error from
// any stage is converted to conditions and forwarded to StatusWriter so the
// Route CR always reflects the latest outcome, even on failure.
func (s *RouteService) Reconcile(ctx context.Context, resolved *routeResolution.ResolvedRoute) error {
	route := s.mapper.MapResolvedToDomain(resolved)

	// A selector failure (unknown runtime) is a terminal outcome — surface it
	// as a Failed condition rather than returning bare.
	provider, selErr := s.backend.ForRoute(route)
	if selErr != nil {
		conditions := s.routeConditions(route, domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: selErr.Error(),
		}, selErr)
		return s.status.Write(ctx, resolved.Route, conditions...)
	}

	result, ensureErr := provider.Ensure(ctx, resolved, route)
	conditions := s.routeConditions(route, result, ensureErr)
	return s.status.Write(ctx, resolved.Route, conditions...)
}

// routeConditions derives a metav1.Condition slice from the domain Route,
// RouteResult, and any provider error.
//
// Conditions are runtime-agnostic — the Ready message reports the resolved
// address rather than naming a specific runtime resource (DomainMapping,
// Ingress, etc). This keeps the condition stable across runtime migrations.
func (s *RouteService) routeConditions(route domain.Route, result domain.RouteResult, runErr error) []metav1.Condition {
	now := metav1.NewTime(time.Now())

	switch {
	case runErr != nil:
		return []metav1.Condition{{
			Type:               "RouteFailed",
			Status:             metav1.ConditionFalse,
			Reason:             "RouteFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}}
	case result.Phase == domain.PhaseReady:
		return []metav1.Condition{{
			Type:               "RouteReady",
			Status:             metav1.ConditionTrue,
			Reason:             "RouteReady",
			Message:            fmt.Sprintf("route active; serving traffic at %s", result.ResolvedAddress),
			LastTransitionTime: now,
		}}
	case result.Phase == domain.PhaseDegraded:
		return []metav1.Condition{{
			Type:               "RouteDegraded",
			Status:             metav1.ConditionFalse,
			Reason:             "RouteDegraded",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	default:
		return []metav1.Condition{{
			Type:               "RoutePending",
			Status:             metav1.ConditionFalse,
			Reason:             "RoutePending",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	}
}
