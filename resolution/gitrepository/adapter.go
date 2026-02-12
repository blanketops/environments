package gitrepository

import (
	"context"

	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
)

type Adapter struct {
	// future deps:
	// client client.Client
	// log    logr.Logger
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Resolve(
	ctx context.Context,
	repo *sourcesv1alpha1.GitRepository,
) (*ResolvedGitRepository, error) {
	return ResolveGitRepository(repo)
}
