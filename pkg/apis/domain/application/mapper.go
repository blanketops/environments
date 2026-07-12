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
This file owns Mapper — the translation layer between the resolved Domain
contract and the domain.Domain aggregate consumed by the provider layer.

The Mapper enforces the resolution contract: it panics on fields the resolver
guarantees to be present (Host, TLSStrategy, RouteRef.Name) so that resolver
bugs surface loudly rather than silently provisioning the wrong cert chain or
creating a ClusterDomainClaim for an empty host.

All other fields are mapped verbatim. MTLS is mapped from the resolved
ResolvedDomainMTLS struct to the flat domain.MTLS value. RenewBefore is
passed through as a raw duration string — the provider parses it.
*/
package application

import (
	"fmt"

	"github.com/blanketops/environments/pkg/apis/domain/domain"
	domainResolution "github.com/blanketops/environments/resolution/domain"
)

// Mapper translates a ResolvedDomain into a domain.Domain aggregate.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved Domain into a domain.Domain
// for consumption by the provider layer.
//
// Panics on resolver invariant violations (empty Host, nil TLSStrategy, empty
// RouteRef.Name) — these indicate a resolver bug, not a user error, and must
// not be silently swallowed. All other fields are mapped verbatim.
func (Mapper) MapResolvedToDomain(rd *domainResolution.ResolvedDomain) domain.Domain {
	spec := rd.Spec

	// Resolver invariants — panic loudly on violation.
	if spec.Host == "" {
		panic(fmt.Sprintf("resolved domain %q has empty Host (resolver bug)", rd.Domain.Name))
	}
	if spec.TLSStrategy == nil {
		panic(fmt.Sprintf("resolved domain %q has nil TLSStrategy (resolver bug)", rd.Domain.Name))
	}
	if spec.RouteRef.Name == "" {
		panic(fmt.Sprintf("resolved domain %q has empty RouteRef.Name (resolver bug)", rd.Domain.Name))
	}

	return domain.Domain{
		Name:        rd.Domain.Name,
		Namespace:   rd.Domain.Namespace,
		Host:        spec.Host,
		RouteRef:    spec.RouteRef.Name,
		TLSStrategy: domain.TLSStrategy(string(*spec.TLSStrategy)),
		MTLS: domain.MTLS{
			Enforced: spec.MTLS.Enforced,
		},
		RenewBefore: spec.RenewBefore,
	}
}
