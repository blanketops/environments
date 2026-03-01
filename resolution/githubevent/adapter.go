package githubevent

import (
	"context"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
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
	githubEvent *eventsv1alpha1.GitHubEvent,
) (*ResolvedGitHubEvent, error) {
	return ResolveGitHubEvent(githubEvent)
}
