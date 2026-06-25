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
This file owns the Domain domain aggregate and the TLSStrategy + MTLS types.

Domain is the authoritative in-memory representation of a Domain CR after
resolution. All downstream application and provider logic operates exclusively
on this type — no raw strings, no re-reads of the Kubernetes CR.

The Domain aggregate owns the cert/mapping chain for a host:
  - platform strategy: DomainClaim only (DNS01 wildcard already issued)
  - custom strategy:   Issuer + DomainClaim + Certificate (HTTP01 ACME)

The DomainMapping that binds the host to the Knative Service is owned by the
Route provider (pkg/routes/api/knative.go). Domain owns the TLS and DNS layers
on top of that binding.

See also:
  - pkg/domain/domain/state.go     — Phase type and constants
  - pkg/domain/domain/result.go    — DomainResult returned by the provider
  - pkg/domain/application/mapper.go — constructs Domain from ResolvedDomainSpec
  - pkg/domain/api/provider.go     — Provider interface consuming Domain
*/
package domain

// Domain is the authoritative domain aggregate for the Domain CR.
// Constructed once by the mapper; consumed by the application service and
// Knative provider. Never mutated after construction.
type Domain struct {
	// Name is the Domain CR name.
	Name string

	// Namespace is the Domain CR namespace.
	Namespace string

	// Host is the FQDN this domain assignment covers.
	Host string

	// RouteRef is the name of the Route CR in the same namespace that owns
	// this Domain. Used for owner-reference validation and log context.
	RouteRef string

	// TLSStrategy is the cert provisioning path for this domain.
	TLSStrategy TLSStrategy

	// MTLS carries the inter-service mTLS configuration.
	// When Enforced is true, blanketops-proxy sidecars are injected and
	// blanketK issues identities for all workloads bound to this domain.
	MTLS MTLS

	// RenewBefore is the cert renewal window (e.g. "720h").
	// Applies to custom strategy only — empty means cert-manager default.
	RenewBefore string
}

// TLSStrategy identifies the cert provisioning path for a Domain.
// Mirrors blanketops.common.v1.DomainTLSStrategy in the proto contract.
type TLSStrategy string

const (
	// TLSStrategyPlatform uses the platform wildcard cert.
	// The DNS01 ClusterIssuer has already issued *.dev.host — the controller
	// emits DomainClaim only. No Certificate work is needed.
	TLSStrategyPlatform TLSStrategy = "platform"

	// TLSStrategyCustom targets a client-owned zone.
	// The controller emits an Issuer + DomainClaim + Certificate.
	// HTTP01 ACME challenge is handled by the nginx solver in the tenant namespace.
	TLSStrategyCustom TLSStrategy = "custom"
)

// MTLS carries the inter-service mTLS configuration for a Domain.
type MTLS struct {
	// Enforced is true when blanketops-proxy + blanketK mTLS is active.
	// The platform wires the sidecar identity — no manual cert management needed.
	Enforced bool
}
