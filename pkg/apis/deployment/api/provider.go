/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package api owns the infra that actually materializes a DeploymentIntent
onto a backend — it does not decide which backend or strategy to use (that
dispatch lives in pkg/apis/deployment/strategy and pkg/apis/deployment/reconcile).

K8SProvider applies/tears down Kubernetes Deployment and Service objects via
server-side apply. KustomizeStrategyProvider takes the GitOps path instead:
it renders manifests (via render/builders), commits them to a repo, and
ensures the Flux GitRepository/Kustomization CRs that make the cluster
reconcile them. Provider and ProviderRegistry describe the interface every
runtime-specific implementation conforms to, so a caller can resolve one by
intent.Runtime without knowing the concrete type.
*/
package api

import (
	"context"
	"fmt"

	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

// Provider executes a DeploymentIntent against a specific runtime backend.
type Provider interface {
	// Runtime returns the runtime this provider supports.
	Runtime() intent.Runtime

	// Supports reports whether the provider supports the given strategy.
	Supports(strategy intent.Strategy) bool

	// Execute reconciles the DeploymentIntent.
	Execute(
		ctx context.Context,
		intent *intent.DeploymentIntent,
	) (*domain.DeploymentResult, error)
}

// ProviderRegistry resolves Providers by runtime.
type ProviderRegistry struct {
	providers map[intent.Runtime]Provider
}

// NewProviderRegistry constructs a ProviderRegistry pre-populated with the
// given providers, indexed by their own Runtime().
func NewProviderRegistry(providers ...Provider) *ProviderRegistry {
	m := make(map[intent.Runtime]Provider)

	for _, p := range providers {
		m[p.Runtime()] = p
	}

	return &ProviderRegistry{
		providers: m,
	}
}

// Register allows late registration (optional).
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Runtime()] = provider
}

// Resolve returns a provider for the given runtime.
func (r *ProviderRegistry) Resolve(
	runtime intent.Runtime,
) (Provider, error) {

	p, ok := r.providers[runtime]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}

	return p, nil
}
