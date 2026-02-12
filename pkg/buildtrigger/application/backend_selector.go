// pkg/buildtrigger/application/backend_selector.go
package application

import (
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/domain"
)

// BackendSelector selects the provider responsible for evaluating BuildTriggers.
//
// IMPORTANT:
// - Selection is PURE
// - No side effects
// - No validation
// - No fallthrough logic hidden in providers
type BackendSelector struct {
	GitHub api.Provider
	Manual api.Provider
	// GitLab api.Provider (future)
}

// NewBackendSelector wires available providers.
func NewBackendSelector(
	github api.Provider,
	manual api.Provider,
) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
		Manual: manual,
	}
}

// ForTrigger returns the provider that can evaluate this trigger.
func (b *BackendSelector) ForTrigger(
	trigger domain.BuildTrigger,
) api.Provider {

	switch trigger.Trigger.Source {

	case domain.TriggerSourceGitHub:
		return b.GitHub

	case domain.TriggerSourceManual:
		return b.Manual

	default:
		// Defensive default — domain should already validate this
		return b.Default()
	}
}

// Default returns the fallback provider.
func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
