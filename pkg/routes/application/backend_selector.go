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
to the correct build provider (Knative, KubernetesIngress, or GatewayAPI).

Selection is driven by the Runtime field. Unlike the build selector — which
falls back to Buildah for unrecognised strategy names — an unknown route
runtime is a malformed CR, not a soft default. ErrRuntimeUnknown is returned
so the controller surfaces it as a Failed condition rather than silently
materialising the wrong backend.

BackendSelector sits in the application layer — it is called by RouteService
after resolution and before provider dispatch.
*/
package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/routes/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/routes/domain"
)

// BackendSelector routes a domain.Route to the correct Provider implementation
// based on the Runtime field. All registered providers must be non-nil at
// construction time.
type BackendSelector struct {
	Knative api.Provider
}

// NewBackendSelector constructs a BackendSelector with the registered route
// providers. v1 registers Knative only; KubernetesIngress and GatewayAPI are
// added here as they land.
func NewBackendSelector(knative api.Provider) *BackendSelector {
	return &BackendSelector{Knative: knative}
}

// ForRoute returns the Provider that should materialize the given Route.
// Returns ErrRuntimeUnknown for any runtime not registered at construction.
func (b *BackendSelector) ForRoute(route domain.Route) (api.Provider, error) {
	switch route.Runtime {
	case domain.RuntimeKnativeService:
		return b.Knative, nil
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrRuntimeUnknown, route.Runtime)
	}
}
