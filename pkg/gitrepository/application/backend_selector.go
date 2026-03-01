package application

import (
	"strings"

	"github.com/ntlaletsi70/blanketops-environments/pkg/gitrepository/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/gitrepository/domain"
)

// BackendSelector selects the appropriate repository provider.
type BackendSelector struct {
	GitHub api.Provider
	// Future:
	// GitLab api.Provider
	// Bitbucket api.Provider
}

func NewBackendSelector(
	github api.Provider,
) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
	}
}

// ForRepository selects a provider based on repository spec.
func (b *BackendSelector) ForRepository(
	spec domain.GitRepository,
) api.Provider {

	switch strings.ToLower(string(spec.Provider)) {
	case "github":
		return b.GitHub

	default:
		// safe fallback: GitHub
		return b.GitHub
	}
}
