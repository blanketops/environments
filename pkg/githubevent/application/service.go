package application

import (
	"context"

	githubeventResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/githubevent"
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
