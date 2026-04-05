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

	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// GitHubEventService orchestrates resolved GitHubEvent reconciliation.
type GitHubEventService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

func NewGitHubEventService(
	mapper *Mapper,
	status *StatusWriter,
	backend *BackendSelector,
) *GitHubEventService {
	return &GitHubEventService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

func (s *GitHubEventService) Reconcile(
	ctx context.Context,
	resolved *githubeventResolution.ResolvedGitHubEvent,
) error {

	// ------------------------------------------------
	// 1. Map → domain envelope
	// ------------------------------------------------
	event := s.mapper.MapResolvedToDomain(resolved)

	// ------------------------------------------------
	// 2. Select backend
	// ------------------------------------------------
	provider := s.backend.Default()

	// ------------------------------------------------
	// 3. Execute backend logic
	// ------------------------------------------------
	result, err := provider.Ensure(ctx, resolved, event)

	// ------------------------------------------------
	// 4. Write status (against CR, single authority)
	// ------------------------------------------------
	return s.status.Write(ctx, resolved.Event, result, err)
}
