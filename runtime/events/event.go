package events

import "time"

type Source string
type EventType string

const (
	SourceGitHub Source = "github"

	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
	EventManual      EventType = "manual"
)

type Repository struct {
	Owner string
	Name  string
}

type RuntimeEvent struct {
	Source     Source
	Type       EventType
	EventID    string // idempotency key
	Repository Repository

	Ref string
	SHA string

	Actor string

	Payload    []byte // raw payload (optional)
	OccurredAt time.Time
}
