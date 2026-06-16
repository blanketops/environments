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
This file owns BuildTriggerService — the application service that orchestrates
BuildTrigger evaluation.

The pipeline is intentionally minimal: map, evaluate, write. The service does
not make admission decisions itself — that belongs to the Provider. It does not
dispatch BuildRuns — that belongs to the build domain after acceptance. It only
sequences the three steps and ensures the BuildTrigger CR always reflects the
outcome.
*/
package application

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
)

// BuildTriggerService orchestrates the BuildTrigger evaluation pipeline.
// It is stateless beyond its collaborators and safe for concurrent use.
type BuildTriggerService struct {
	mapper   *Mapper
	status   *StatusWriter
	provider api.Provider
}

// NewBuildTriggerService constructs a BuildTriggerService with the required
// collaborators. Provider is a single instance — BackendSelector routes to the
// correct provider before construction if multiple sources are supported.
func NewBuildTriggerService(
	mapper *Mapper,
	status *StatusWriter,
	provider api.Provider,
) *BuildTriggerService {
	return &BuildTriggerService{
		mapper:   mapper,
		status:   status,
		provider: provider,
	}
}

// Evaluate maps the resolved trigger to a domain model, delegates admission
// to the provider, then writes the decision back to the BuildTrigger CR.
// Status is always written — even when evaluation fails — so the CR reflects
// the latest outcome.
func (s *BuildTriggerService) Evaluate(
	ctx context.Context,
	resolved *buildtriggerResolution.ResolvedBuildTrigger,
) error {
	// Map resolved contract → domain trigger.
	trigger := s.mapper.MapResolvedToDomain(resolved)

	// Evaluate — pure decision, no side effects.
	decision, err := s.provider.Evaluate(ctx, resolved, trigger)

	// Write outcome to the BuildTrigger CR regardless of error.
	return s.status.Write(
		ctx,
		resolved.Trigger,
		domain.BuildTriggerStatus{Accepted: decision.Accepted},
		err,
	)
}
