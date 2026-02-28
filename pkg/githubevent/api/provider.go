package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/githubevent/domain"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// Provider realizes GitHubEvent observability infrastructure.

type Provider interface {
	Ensure(
		ctx context.Context,
		resolved *githubeventResolution.ResolvedGitHubEvent,
		event domain.GitHubEvent,
	) (domain.GitHubEventResult, error)
}
