package application

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
)

type GitOpsDecorator struct {
	inner api.Provider
}

func NewGitOpsDecorator(inner api.Provider) api.Provider {
	return &GitOpsDecorator{
		inner: inner,
	}
}

func (g *GitOpsDecorator) Runtime() intent.Runtime {
	return g.inner.Runtime()
}

func (g *GitOpsDecorator) Supports(s intent.Strategy) bool {
	return g.inner.Supports(s)
}

func (g *GitOpsDecorator) Execute(
	ctx context.Context,
	i *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	// Commit manifests to repo / trigger Flux here

	return g.inner.Execute(ctx, i)
}
