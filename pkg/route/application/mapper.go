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
This file owns the Mapper — the translation layer between the resolved Route
contract and the domain Route aggregate consumed by the provider layer.

The Mapper enforces the resolution contract: it panics on fields the resolver
guarantees to be present (Host, Runtime) so that resolver bugs surface loudly
rather than silently producing an empty DomainMapping.

ServiceRef is not a resolver invariant — it is read from the
environments.blanketops.dev/service-unit label that the controller stamps on
the Route CR, and is passed through verbatim. The Knative provider guards an
empty ServiceRef with ErrServiceRefEmpty at Ensure time; the Mapper never
invents defaults or modifies intent.
*/
package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/route/domain"
	routeResolution "github.com/ntlaletsi70/blanketops-environments/resolution/route"
)

// labelServiceUnit is the Route CR label carrying the Knative Service name
// this route exposes. Stamped by the controller during Environment deployment.
const labelServiceUnit = "environments.blanketops.dev/service-unit"

// Mapper translates a ResolvedRoute into a domain.Route.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved Route into a domain Route
// for consumption by the provider layer.
//
// Panics on resolver invariant violations (empty Host or nil Runtime) — these
// indicate a resolver bug, not a user error, and must not be silently
// swallowed. All other fields are mapped verbatim.
//
// ServiceRef is read from the environments.blanketops.dev/service-unit label.
// It may be empty here — the Knative provider enforces its presence via
// ErrServiceRefEmpty when Runtime is RuntimeKnativeService.
func (Mapper) MapResolvedToDomain(rr *routeResolution.ResolvedRoute) domain.Route {
	spec := rr.Spec

	// Resolver invariants — panic loudly on violation.
	if spec.Host == "" {
		panic(fmt.Sprintf("resolved route %q has empty Host (resolver bug)", rr.Route.Name))
	}
	if spec.Runtime == nil {
		panic(fmt.Sprintf("resolved route %q has nil Runtime (resolver bug)", rr.Route.Name))
	}

	// ServiceRef is optional at map time — stamped as a label by the controller.
	var serviceRef string
	if rr.Route.Labels != nil {
		serviceRef = rr.Route.Labels[labelServiceUnit]
	}

	return domain.Route{
		Name:       rr.Route.Name,
		Namespace:  rr.Route.Namespace,
		Host:       spec.Host,
		Enabled:    spec.Enabled,
		Path:       spec.Path,
		TLSEnabled: spec.TLSEnabled,
		Runtime:    domain.Runtime(string(*spec.Runtime)),
		ServiceRef: serviceRef,
	}
}
