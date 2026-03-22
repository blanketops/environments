package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
)

// BackendSelector resolves and prepares a Provider
// for a given DeploymentIntent.
type BackendSelector struct {
	registry *api.ProviderRegistry
}

// NewBackendSelector wires the selector with a ProviderRegistry.
func NewBackendSelector(
	registry *api.ProviderRegistry,
) *BackendSelector {

	return &BackendSelector{
		registry: registry,
	}
}

// Resolve returns a fully prepared Provider for the given intent.
// It validates runtime + strategy compatibility and applies
// optional delivery decorators (e.g., GitOps).
func (b *BackendSelector) Resolve(
	deploymentIntent *intent.DeploymentIntent,
) (api.Provider, error) {

	if b.registry == nil {
		return nil, fmt.Errorf("provider registry is not initialized")
	}

	// 1️⃣ Resolve runtime
	provider, err := b.registry.Resolve(deploymentIntent.Runtime)
	if err != nil {
		return nil, err
	}

	// 2️⃣ Validate strategy support
	if !provider.Supports(deploymentIntent.Strategy) {
		return nil, fmt.Errorf(
			"strategy %s not supported for runtime %s",
			deploymentIntent.Strategy,
			deploymentIntent.Runtime,
		)
	}

	// 3️⃣ Apply GitOps decoration if manifests repo defined
	if deploymentIntent.ManifestsRepo != nil {
		provider = NewGitOpsDecorator(provider)
	}

	return provider, nil
}
