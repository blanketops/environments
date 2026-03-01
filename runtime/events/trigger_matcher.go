package events

import "context"

// ResolvedGitHubEvent comes from resolution/githubevent
type ResolvedGitHubEvent interface {
	EventType() EventType
	Repository() Repository
	Ref() string
	SHA() string
	Actor() string
}

// ResolvedBuildTrigger comes from resolution/buildtrigger
type ResolvedBuildTrigger interface {
	Name() string
}

type BuildTriggerMatcher interface {
	Match(
		ctx context.Context,
		event ResolvedGitHubEvent,
	) ([]ResolvedBuildTrigger, error)
}
