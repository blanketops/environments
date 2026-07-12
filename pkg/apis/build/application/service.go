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
This file owns BuildService — the application service that orchestrates the
build reconciliation pipeline.

BuildService is the single entry point called by the build domain after
resolution. It sequences four steps in order:

 1. Map the resolved Build contract to a domain BuildSpec (Mapper).
 2. Select the correct build provider for the strategy (BackendSelector).
 3. Dispatch execution via the provider (Provider.Run).
 4. Write the outcome back to the Build CR status (StatusWriter).

BuildService does not own retry logic, trigger evaluation, or condition
management — those belong to the domain and observer layers respectively.
*/
package application

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blanketops/environments/pkg/apis/build/domain"
	bldResolution "github.com/blanketops/environments/resolution/build"
)

// BuildService orchestrates the build reconciliation pipeline.
// It is stateless beyond its collaborators and safe for concurrent use.
type BuildService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

// NewBuildService constructs a BuildService with the required collaborators.
func NewBuildService(mapper *Mapper, status *StatusWriter, backend *BackendSelector) *BuildService {
	return &BuildService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

// Reconcile executes the full build pipeline for a resolved Build CR.
// It maps, selects, executes, and writes status in sequence. An error from
// any stage is forwarded to StatusWriter so the Build CR always reflects the
// latest outcome, even on failure.
func (s *BuildService) Reconcile(ctx context.Context, resolved *bldResolution.ResolvedBuild) error {
	spec := s.mapper.MapResolvedToDomain(resolved)
	provider := s.backend.ForSpec(spec)
	result, err := provider.Run(ctx, resolved, spec)

	// Build conditions from outcome. This is the switch, moved from StatusWriter.
	conditions := s.buildConditions(result, err)

	// Pass clean conditions to the dumb writer.
	return s.status.Write(ctx, resolved.Build, conditions...)
}

// Teardown reverses Reconcile — maps the resolved Build, selects the same
// backend provider strategy would have used, and dispatches Teardown to
// delete the owned BuildRun(s) and Build.
func (s *BuildService) Teardown(ctx context.Context, resolved *bldResolution.ResolvedBuild) error {
	spec := s.mapper.MapResolvedToDomain(resolved)
	provider := s.backend.ForSpec(spec)
	return provider.Teardown(ctx, resolved)
}

// buildConditions derives metav1.Condition slice from BuildResult + error.
// This is application logic: it knows what Success/Triggered/Error mean.
func (s *BuildService) buildConditions(result domain.BuildResult, runErr error) []metav1.Condition {
	now := metav1.NewTime(time.Now())

	switch {
	case runErr != nil:
		return []metav1.Condition{{
			Type:               "BuildFailed",
			Status:             metav1.ConditionFalse,
			Reason:             "BuildFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}}
	case result.Success:
		return []metav1.Condition{{
			Type:               "BuildSuccess",
			Status:             metav1.ConditionTrue,
			Reason:             "BuildSucceeded",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	case result.Triggered:
		return []metav1.Condition{{
			Type:               "BuildReady",
			Status:             metav1.ConditionTrue,
			Reason:             "BuildReady",
			Message:            "Build dispatched",
			LastTransitionTime: now,
		}}
	default:
		return []metav1.Condition{{
			Type:               "BuildPending",
			Status:             metav1.ConditionFalse,
			Reason:             "BuildPending",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	}
}
