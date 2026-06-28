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
This file owns Mapper — the translation layer between the resolved Route
contract and the domain Route aggregate consumed by the provider layer.

The Mapper enforces the resolution contract: it panics on fields the resolver
guarantees to be present (Host, Runtime, ServiceUnitRef) so that resolver bugs
surface loudly rather than silently producing an empty DomainMapping or Ingress.

ServiceRef is derived directly from spec.ServiceUnitRef.Name — the resolver
validates this as a required field. The controller convention enforces:

	ksvc name == ServiceUnit name

No label lookup, no API server read, no status lookup required.
*/
package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/route/domain"
	routeResolution "github.com/ntlaletsi70/blanketops-environments/resolution/route"
)

// Mapper translates a ResolvedRoute into a domain.Route.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved Route into a domain Route
// for consumption by the provider layer.
//
// Panics on resolver invariant violations (empty Host, nil Runtime, nil
// ServiceUnitRef) — these indicate a resolver bug, not a user error, and
// must not be silently swallowed. All other fields are mapped verbatim.
//
// ServiceRef is derived from spec.ServiceUnitRef.Name by convention:
//
//	ksvc name == ServiceUnit name
//
// The provider layer guards an empty ServiceRef with ErrServiceRefEmpty
// at Ensure time as a final safety net.
func (Mapper) MapResolvedToDomain(rr *routeResolution.ResolvedRoute) domain.Route {
	spec := rr.Spec

	// Resolver invariants — panic loudly on violation.
	if spec.Host == "" {
		panic(fmt.Sprintf("resolved route %q has empty Host (resolver bug)", rr.Route.Name))
	}
	if spec.Runtime == nil {
		panic(fmt.Sprintf("resolved route %q has nil Runtime (resolver bug)", rr.Route.Name))
	}
	if spec.ServiceUnitRef == nil || spec.ServiceUnitRef.Name == "" {
		panic(fmt.Sprintf("resolved route %q has nil or empty ServiceUnitRef (resolver bug)", rr.Route.Name))
	}

	return domain.Route{
		Name:       rr.Route.Name,
		Namespace:  rr.Route.Namespace,
		Host:       spec.Host,
		Enabled:    spec.Enabled,
		Path:       spec.Path,
		TLSEnabled: spec.TLSEnabled,
		Runtime:    domain.Runtime(string(*spec.Runtime)),
		// Convention: ksvc name == ServiceUnit name.
		// Derived from the resolved ServiceUnitRef — no label, no API server read.
		ServiceRef: spec.ServiceUnitRef.Name,
	}
}
