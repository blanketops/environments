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
This file owns BackendSelector — the routing layer that maps a domain.Domain
to the correct Provider implementation based on TLSStrategy.

Unlike the route BackendSelector which maps Runtime to distinct provider
implementations, the domain BackendSelector maps TLSStrategy to a single
KnativeProvider — strategy dispatch (platform vs custom) is handled internally
by the provider. The BackendSelector exists for consistency with the application
layer pattern and future extensibility (e.g. a GatewayAPI cert provider).

Unknown strategies return ErrStrategyUnknown — the resolver validates strategy
at resolution time, so this is a safety fallback for malformed CRs.

BackendSelector sits in the application layer — it is called by DomainService
after resolution and before provider dispatch.
*/
package application

import (
	"fmt"

	"github.com/BlanketOps/blanketops-environments/pkg/apis/domain/api"
	"github.com/BlanketOps/blanketops-environments/pkg/apis/domain/domain"
)

// BackendSelector maps a domain.Domain to the correct Provider based on TLSStrategy.
// KnativeProvider handles both platform and custom strategies internally via
// its own strategy switch — the BackendSelector is the first line of defence.
type BackendSelector struct {
	Knative api.Provider
}

// NewBackendSelector constructs a BackendSelector with the registered domain provider.
func NewBackendSelector(knative api.Provider) *BackendSelector {
	return &BackendSelector{Knative: knative}
}

// ForDomain returns the Provider for the given Domain.
// Both TLSStrategyPlatform and TLSStrategyCustom map to KnativeProvider.
// Unknown strategies return ErrStrategyUnknown.
func (b *BackendSelector) ForDomain(d domain.Domain) (api.Provider, error) {
	switch d.TLSStrategy {
	case domain.TLSStrategyPlatform, domain.TLSStrategyCustom:
		return b.Knative, nil
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrStrategyUnknown, d.TLSStrategy)
	}
}
