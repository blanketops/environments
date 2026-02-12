package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/intent"
)

// Provider executes a DeploymentIntent against a runtime backend.
type Provider interface {
	Execute(
		ctx context.Context,
		intent *intent.DeploymentIntent,
	) (*domain.DeploymentResult, error)
}

// ProviderRegistry resolves Providers by runtime.
type ProviderRegistry struct {
	providers map[intent.Runtime]Provider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[intent.Runtime]Provider),
	}
}

func (r *ProviderRegistry) Register(
	runtime intent.Runtime,
	provider Provider,
) {
	r.providers[runtime] = provider
}

func (r *ProviderRegistry) Get(
	runtime intent.Runtime,
) (Provider, bool) {
	p, ok := r.providers[runtime]
	return p, ok
}
