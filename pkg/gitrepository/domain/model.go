package domain

// Provider identifies a supported VCS provider.
type Provider string

const (
	ProviderGitHub Provider = "github"
	// Future:
	// ProviderGitLab Provider = "gitlab"
	// ProviderBitbucket Provider = "bitbucket"
)

// EventType represents a VCS event the system can subscribe to.
type EventType string

const (
	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
)

// RepositoryID is the external identity of a repository.
type RepositoryID struct {
	Owner string
	Name  string
}

// IsValid validates the repository identity.
func (r RepositoryID) IsValid() bool {
	return r.Owner != "" && r.Name != ""
}

// WebhookSpec declares which events the repository should emit.
type WebhookSpec struct {
	Events []EventType
}

// IsValid validates the webhook specification.
func (w WebhookSpec) IsValid() bool {
	return len(w.Events) != 0
}

// GitRepository is the domain model representing a registered source repository.
type GitRepository struct {
	Provider   Provider
	Repository RepositoryID
	Webhooks   []WebhookSpec
}

// Validate enforces domain invariants.
func (g GitRepository) Validate() error {
	if g.Provider == "" {

		return ErrMissingProvider
	}

	if !g.Repository.IsValid() {
		return ErrInvalidRepository
	}

	for _, wh := range g.Webhooks {
		if !wh.IsValid() {
			return ErrInvalidWebhook
		}
	}

	return nil
}
