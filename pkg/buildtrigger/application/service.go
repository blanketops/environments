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

package application

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
)

type BuildTriggerService struct {
	mapper   *Mapper
	status   *StatusWriter
	provider api.Provider
}

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

func (s *BuildTriggerService) Evaluate(
	ctx context.Context,
	resolved *buildtriggerResolution.ResolvedBuildTrigger,
) error {

	// ------------------------------------------------
	// 1. Map → domain trigger (MUST stay)
	// ------------------------------------------------
	trigger := s.mapper.MapResolvedToDomain(resolved)

	// ------------------------------------------------
	// 2. Delegate evaluation (NO recursion)
	// ------------------------------------------------
	decision, err := s.provider.Evaluate(
		ctx,
		resolved,
		trigger,
	)

	// ------------------------------------------------
	// 3. Write status (against BuildTrigger CR)
	// ------------------------------------------------
	return s.status.Write(
		ctx,
		resolved.Trigger,
		domain.BuildTriggerStatus{
			Accepted: decision.Accepted,
		},
		err,
	)
}
