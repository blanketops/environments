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
This file owns BackendSelector — the routing layer that maps a domain.Route
to the correct Provider implementation based on the Runtime field.

Unlike the build selector — which falls back to Buildah for unrecognised
strategy names — an unknown route runtime is a malformed CR, not a soft
default. ErrRuntimeUnknown is returned so the controller surfaces it as a
Failed condition rather than silently materialising the wrong backend.

Registered runtimes:

	RuntimeKnativeService        → KnativeProvider  (Knative DomainMapping via Kourier)
	RuntimeKubernetesContainer   → IngressProvider  (networking.k8s.io/v1 Ingress via nginx)
	RuntimeGatewayAPI            → not yet registered — returns ErrRuntimeUnknown

BackendSelector sits in the application layer — it is called by RouteService
after resolution and before provider dispatch.
*/
package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/route/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/route/domain"
)

// BackendSelector routes a domain.Route to the correct Provider implementation
// based on the Runtime field. All registered providers must be non-nil at
// construction time.
type BackendSelector struct {
	Knative api.Provider
	Ingress api.Provider
}

// NewBackendSelector constructs a BackendSelector with the registered route
// providers. Knative and KubernetesIngress are registered at v1.
// GatewayAPI is added here when it lands.
func NewBackendSelector(knative api.Provider, ingress api.Provider) *BackendSelector {
	return &BackendSelector{
		Knative: knative,
		Ingress: ingress,
	}
}

// ForRoute returns the Provider that should materialize the given Route.
// Returns ErrRuntimeUnknown for any runtime not registered at construction.
func (b *BackendSelector) ForRoute(route domain.Route) (api.Provider, error) {
	switch route.Runtime {
	case domain.RuntimeKnativeService:
		return b.Knative, nil
	case domain.RuntimeKubernetesContainer:
		return b.Ingress, nil
	// RuntimeGatewayAPI: not yet registered — fall through to ErrRuntimeUnknown.
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrRuntimeUnknown, route.Runtime)
	}
}
