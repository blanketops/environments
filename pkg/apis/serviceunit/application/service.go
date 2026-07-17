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
This file owns ServiceUnitService — the application service that orchestrates
the ServiceUnit reconciliation pipeline.

ServiceUnitService is the single entry point called by the ServiceUnit
controller after resolution. Unlike RouteService or BuildService, there is no
BackendSelector/Provider dispatch step: a ServiceUnit does not materialize
anything onto infrastructure itself (a Deployment CR does that). The
pipeline is:

 1. Map the resolved ServiceUnit contract to a domain ServiceUnit (Mapper).
 2. Derive the outcome directly from the mapped fields (deriveResult).
 3. Write the outcome back to the ServiceUnit CR status (StatusWriter).

This write is the authoritative signal downstream consumers (e.g. a
Deployment CR referencing this unit) should read, rather than re-deriving
ServiceUnit state inline from the raw resolved contract themselves.
*/
package application

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blanketops/environments/pkg/apis/serviceunit/domain"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

// ServiceUnitService orchestrates the ServiceUnit reconciliation pipeline.
// It is stateless beyond its collaborators and safe for concurrent use.
type ServiceUnitService struct {
	mapper *Mapper
	status *StatusWriter
}

// NewServiceUnitService constructs a ServiceUnitService with the required collaborators.
func NewServiceUnitService(mapper *Mapper, status *StatusWriter) *ServiceUnitService {
	return &ServiceUnitService{
		mapper: mapper,
		status: status,
	}
}

// Reconcile executes the full ServiceUnit pipeline for a resolved
// ServiceUnit CR. It maps, derives, and writes status in sequence.
func (s *ServiceUnitService) Reconcile(ctx context.Context, resolved *serviceunitResolution.ResolvedServiceUnit) error {
	if resolved == nil {
		return domain.ErrServiceUnitNil
	}

	unit := s.mapper.MapResolvedToDomain(resolved)
	result := deriveResult(unit)
	conditions := serviceUnitConditions(result)

	return s.status.Write(ctx, resolved.ServiceUnit, conditions...)
}

// deriveResult derives the ServiceUnit's outcome directly from its mapped
// fields — there is no provider call to consult, only the resolved contract.
func deriveResult(unit domain.ServiceUnit) domain.Result {
	switch unit.Type {
	case domain.TypeStatic:
		if unit.Image == "" {
			return domain.Result{Phase: domain.PhaseFailed, Message: "static unit has no resolved image (resolver bug)"}
		}
		return domain.Result{Phase: domain.PhaseReady}

	case domain.TypeBuild:
		if unit.Image == "" {
			return domain.Result{Phase: domain.PhasePending, Message: "awaiting image from source Build"}
		}
		return domain.Result{Phase: domain.PhaseReady}

	case domain.TypeSupplyChain:
		return domain.Result{Phase: domain.PhasePending, Message: "supplychain resolution not yet implemented"}

	default:
		return domain.Result{Phase: domain.PhaseFailed, Message: fmt.Sprintf("unknown type %q", unit.Type)}
	}
}

// serviceUnitConditions derives a metav1.Condition slice from the domain Result.
func serviceUnitConditions(result domain.Result) []metav1.Condition {
	now := metav1.NewTime(time.Now())

	switch result.Phase {
	case domain.PhaseReady:
		return []metav1.Condition{{
			Type:               "ServiceUnitReady",
			Status:             metav1.ConditionTrue,
			Reason:             "ServiceUnitReady",
			Message:            "image resolved",
			LastTransitionTime: now,
		}}
	case domain.PhaseFailed:
		return []metav1.Condition{{
			Type:               "ServiceUnitFailed",
			Status:             metav1.ConditionFalse,
			Reason:             "ServiceUnitFailed",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	default:
		return []metav1.Condition{{
			Type:               "ServiceUnitPending",
			Status:             metav1.ConditionFalse,
			Reason:             "ServiceUnitPending",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	}
}
