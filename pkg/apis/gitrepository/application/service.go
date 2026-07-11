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

	gitrepoResolution "github.com/BlanketOps/environments/resolution/gitrepository"
)

// GitRepositoryService orchestrates GitRepository reconciliation.
type GitRepositoryService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

func NewGitRepositoryService(mapper *Mapper, status *StatusWriter, backend *BackendSelector) *GitRepositoryService {
	return &GitRepositoryService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

// Reconcile reconciles a resolved GitRepository declaratively.
func (s *GitRepositoryService) Reconcile(ctx context.Context, resolved *gitrepoResolution.ResolvedGitRepository) error {

	// ------------------------------------------------
	// 1. Map resolved → domain
	// ------------------------------------------------
	domainRepo := s.mapper.MapResolvedToDomain(resolved)

	// ------------------------------------------------
	// 2. Select backend (github, gitlab, etc)
	// ------------------------------------------------
	provider := s.backend.ForRepository(domainRepo)

	// ------------------------------------------------
	// 3. Ensure external state
	// ------------------------------------------------
	result, err := provider.Ensure(ctx, resolved.Repository, domainRepo)

	// ------------------------------------------------
	// 4. Write status (against CR)
	// ------------------------------------------------
	return s.status.Write(ctx, resolved.Repository, result, err)
}
