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
Package domain — contract_adapter.go

This file owns the proto projection for Domain — a one-way mapping from the
typed ResolvedDomainSpec to the canonical proto DomainSpec used for
content-addressable hashing, audit trails, and gRPC API responses.

One-way contract:

	ResolvedDomainSpec → networkscontractv1alpha1.DomainSpec

This projection is NEVER fed back into controllers. It is the canonical
serialization of a resolved Domain intent — output only.

TLS strategy wrapper:

	DomainSpec.TlsStrategy is *commonv1.DomainTLSStrategy — a common/v1 wrapper
	message consistent with the RouteRuntime / RoutePhase wrapper pattern.

	"platform" → DOMAIN_TLS_STRATEGY_PLATFORM
	"custom"   → DOMAIN_TLS_STRATEGY_CUSTOM
	nil / ""   → DOMAIN_TLS_STRATEGY_UNSPECIFIED  (safety fallback only —
	             tlsStrategy is validated as mandatory in resolve.go)

See also:
  - resolution/domain/resolve.go — resolution entry point and TLSStrategy type
  - resolution/domain/adapter.go — interface wrapper
  - blanketops/networks/v1alpha1/domain.proto
  - blanketops/common/v1/domain.proto
*/
package domain

import (
	commonv1 "github.com/BlanketOps/environments-contract/blanketops/common/v1"
	networkscontractv1alpha1 "github.com/BlanketOps/environments-contract/blanketops/networks/v1alpha1"
)

// ToDomainContract projects a ResolvedDomain into the canonical proto DomainSpec
// for infrastructure consumers (hashing, audit logging, gRPC serialization).
// Returns nil, nil for nil input — callers decide whether that is an error.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func ToDomainContract(d *ResolvedDomain) (*networkscontractv1alpha1.DomainSpec, error) {
	if d == nil || d.Spec == nil {
		return nil, nil
	}

	return &networkscontractv1alpha1.DomainSpec{
		Host:        d.Spec.Host,
		RouteRef:    mapRouteRef(d.Spec.RouteRef),
		TlsStrategy: mapTLSStrategy(d.Spec.TLSStrategy),
		Mtls:        mapMTLS(d.Spec.MTLS),
		RenewBefore: d.Spec.RenewBefore,
	}, nil
}

// mapRouteRef converts the resolved route ref to the proto DomainRouteRef message.
func mapRouteRef(ref ResolvedDomainRouteRef) *networkscontractv1alpha1.DomainRouteRef {
	return &networkscontractv1alpha1.DomainRouteRef{
		Name: ref.Name,
	}
}

// mapTLSStrategy converts the local TLSStrategy to the proto *commonv1.DomainTLSStrategy
// wrapper message. Absent or unknown values produce DOMAIN_TLS_STRATEGY_UNSPECIFIED.
// The pre-validated TLSStrategy constant values guarantee the known cases always match.
func mapTLSStrategy(ts *TLSStrategy) *commonv1.DomainTLSStrategy {
	if ts == nil {
		return &commonv1.DomainTLSStrategy{
			Strategy: commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_UNSPECIFIED,
		}
	}
	switch *ts {
	case TLSStrategyPlatform:
		return &commonv1.DomainTLSStrategy{
			Strategy: commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_PLATFORM,
		}
	case TLSStrategyCustom:
		return &commonv1.DomainTLSStrategy{
			Strategy: commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_CUSTOM,
		}
	default:
		return &commonv1.DomainTLSStrategy{
			Strategy: commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_UNSPECIFIED,
		}
	}
}

// mapMTLS converts the resolved mTLS config to the proto DomainMTLS message.
func mapMTLS(m ResolvedDomainMTLS) *networkscontractv1alpha1.DomainMTLS {
	return &networkscontractv1alpha1.DomainMTLS{
		Enforced: m.Enforced,
	}
}
