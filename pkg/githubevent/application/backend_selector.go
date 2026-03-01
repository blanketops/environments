package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
)

// BackendSelector selects the backend responsible for handling GitHub events.
type BackendSelector struct {
	GitHub api.Provider
}

func NewBackendSelector(
	github api.Provider,
) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
	}
}

// ForEvent returns the provider that can handle this event.
// At this stage, all events are GitHub events by definition.
func (b *BackendSelector) ForEvent(_ domain.GitHubEvent) api.Provider {
	return b.GitHub
}

func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
