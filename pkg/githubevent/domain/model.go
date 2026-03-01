package domain

// EventType is the semantic type of a GitHub event.
type EventType string
type EventID string

const (
	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
)

// GitRef represents a Git reference.
type GitRef struct {
	Name string // e.g. refs/heads/main
}

// Commit represents a commit involved in the event.
type Commit struct {
	SHA string
}

// Actor represents the identity that triggered the event.
type Actor struct {
	Login string
}

// Repository identifies the source repository.
type Repository struct {
	FullName string // owner/name
	Owner    string
	Name     string
}

// GitHubEvent is a pure domain fact:
// “Something happened in a repository”.
type GitHubEvent struct {
	EventID    string
	Type       EventType
	Repository Repository
	Ref        GitRef
	Commit     Commit
	Actor      Actor
}

// -----------------------------------------------------------------------------
// GitHubEventStatus
// -----------------------------------------------------------------------------
//
// Internal domain status for a GitHubEvent decision.
// This is serialized into CRD status.contract.
type GitHubEventStatus struct {
	// Was the event accepted as valid?
	Accepted bool

	// Did this event trigger downstream work?
	Triggered bool

	// Human-readable explanation
	Message string

	// Reference to triggered resource (optional)
	TriggeredRef string
}
