package application

import (
	"context"

	gitrepoResolution "github.com/ntlaletsi70/blanketops-environments/resolution/gitrepository"
)

// GitRepositoryService orchestrates GitRepository reconciliation.
type GitRepositoryService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

func NewGitRepositoryService(
	mapper *Mapper,
	status *StatusWriter,
	backend *BackendSelector,
) *GitRepositoryService {
	return &GitRepositoryService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

// Reconcile reconciles a resolved GitRepository declaratively.
func (s *GitRepositoryService) Reconcile(
	ctx context.Context,
	resolved *gitrepoResolution.ResolvedGitRepository,
) error {

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
