package api

import (
	"context"

	githubeventResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/githubevent"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/githubevent/domain"
)

// Provider realizes GitHubEvent observability infrastructure.

type Provider interface {
	Ensure(
		ctx context.Context,
		resolved *githubeventResolution.ResolvedGitHubEvent,
		event domain.GitHubEvent,
	) (domain.GitHubEventResult, error)
}
