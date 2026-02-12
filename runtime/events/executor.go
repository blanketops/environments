package events

import "context"

type Executor interface {
	Execute(
		ctx context.Context,
		trigger ResolvedBuildTrigger,
		event ResolvedGitHubEvent,
	) error
}
