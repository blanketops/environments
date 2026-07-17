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
Package resolve implements resolution for the Domain CR.

The Domain CR stores its spec as a raw JSON contract (spec.contract).
ResolveDomain decodes this into a fully typed ResolvedDomain — the authoritative
runtime representation consumed by all downstream domain and application logic.

Domain owns the cert/mapping chain for a host within a BlanketOps Environment.
It is owned by a Route CR via ownerReference (cascade delete). Route owns the
workload binding (runtime, path, enabled); Domain owns TLS and mTLS identity.

Required contract fields:
  - host        — the FQDN this domain covers
  - routeRef    — the Route CR in this namespace that owns this Domain
  - tlsStrategy — the cert provisioning path (platform | custom)

Optional contract fields:
  - mtls.enforced — whether mTLS is enforced via blanketops-proxy + blanketK
  - renewBefore   — cert renewal window; only applies to tlsStrategy: custom

TLS strategy semantics:

	platform: Platform wildcard cert (*.dev.host) already covers this host via a
	          DNS01 ClusterIssuer. Controller emits DomainClaim + DomainMapping
	          only. renewBefore is ignored.

	custom:   Client-owned zone (e.g. client-a.co.za). Controller emits
	          Issuer + DomainClaim + DomainMapping + Certificate. HTTP01 ACME
	          challenge via the nginx solver in the tenant namespace.

All failures surface as errors. Resolution never panics.
*/
package resolve

import (
	"encoding/json"
	"fmt"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
)

// -----------------------------------------------------------------------------
// TLSStrategy type
// -----------------------------------------------------------------------------

// TLSStrategy identifies the cert provisioning path for a Domain.
// Corresponds to blanketops.common.v1.DomainTLSStrategy in the proto contract.
type TLSStrategy string

const (
	// TLSStrategyPlatform uses the platform wildcard cert.
	// DNS01 ClusterIssuer already issued — DomainClaim + DomainMapping only.
	TLSStrategyPlatform TLSStrategy = "platform"

	// TLSStrategyCustom targets a client-owned zone.
	// Controller emits Issuer + DomainClaim + DomainMapping + Certificate.
	TLSStrategyCustom TLSStrategy = "custom"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedDomain pairs the original Kubernetes Domain object with its fully
// decoded and validated spec.
type ResolvedDomain struct {
	Domain *networksv1alpha1.Domain
	Spec   *ResolvedDomainSpec
}

// ResolvedDomainSpec is the decoded and validated Domain spec.
// Fields mirror the proto DomainSpec — consumers must not re-read from the raw CR.
type ResolvedDomainSpec struct {
	// Host is the FQDN this domain assignment covers.
	Host string

	// RouteRef identifies the Route CR in the same namespace that owns this Domain.
	RouteRef ResolvedDomainRouteRef

	// TLSStrategy is the cert provisioning path (platform | custom).
	// Mandatory — the downstream cert and DomainMapping logic branches on this.
	TLSStrategy *TLSStrategy

	// MTLS carries inter-service mTLS configuration for this domain.
	// Absent or not enforced means plain TLS only.
	MTLS ResolvedDomainMTLS

	// RenewBefore is the cert renewal window (e.g. "720h").
	// Applies to custom strategy only — empty means cert-manager default.
	RenewBefore string
}

// ResolvedDomainRouteRef is the resolved Route reference.
type ResolvedDomainRouteRef struct {
	// Name is the Route CR name in the same namespace as this Domain.
	Name string
}

// ResolvedDomainMTLS carries the resolved mTLS configuration.
type ResolvedDomainMTLS struct {
	// Enforced is true when blanketops-proxy + blanketK mTLS is active.
	Enforced bool
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveDomain decodes and validates the raw JSON contract from the Domain CR
// spec into a ResolvedDomain. Returns an error if the CR is nil, the contract
// is absent, or any required field is missing or malformed.
func ResolveDomain(domain *networksv1alpha1.Domain) (*ResolvedDomain, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain is nil")
	}
	if len(domain.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(domain.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Required fields.
	// ------------------------------------------------

	host, err := requireString(raw, "host")
	if err != nil {
		return nil, err
	}

	routeRef, err := resolveRouteRef(raw)
	if err != nil {
		return nil, err
	}

	tlsStrategyStr, err := requireString(raw, "tlsStrategy")
	if err != nil {
		return nil, err
	}
	if tlsStrategyStr != string(TLSStrategyPlatform) && tlsStrategyStr != string(TLSStrategyCustom) {
		return nil, fmt.Errorf("field \"tlsStrategy\" must be %q or %q, got %q",
			TLSStrategyPlatform, TLSStrategyCustom, tlsStrategyStr)
	}
	ts := TLSStrategy(tlsStrategyStr)

	// ------------------------------------------------
	// Optional fields.
	// ------------------------------------------------

	// mtls: absent means mTLS is not enforced.
	mtls, err := resolveMTLS(raw)
	if err != nil {
		return nil, err
	}

	// renewBefore: absent means cert-manager default applies.
	// Only meaningful for tlsStrategy: custom.
	renewBefore := optionalString(raw, "renewBefore")

	return &ResolvedDomain{
		Domain: domain,
		Spec: &ResolvedDomainSpec{
			Host:        host,
			RouteRef:    routeRef,
			TLSStrategy: &ts,
			MTLS:        mtls,
			RenewBefore: renewBefore,
		},
	}, nil
}

// -----------------------------------------------------------------------------
// Nested object resolution
// -----------------------------------------------------------------------------

// resolveRouteRef extracts and validates the mandatory routeRef block.
func resolveRouteRef(raw map[string]any) (ResolvedDomainRouteRef, error) {
	var ref ResolvedDomainRouteRef

	v, ok := raw["routeRef"]
	if !ok {
		return ref, fmt.Errorf("missing required field \"routeRef\"")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return ref, fmt.Errorf("field \"routeRef\" must be an object")
	}

	name, err := requireString(m, "name")
	if err != nil {
		return ref, fmt.Errorf("routeRef.name: %w", err)
	}

	ref.Name = name
	return ref, nil
}

// resolveMTLS extracts the optional mtls block.
// Returns a zero ResolvedDomainMTLS (Enforced: false) when the field is absent.
func resolveMTLS(raw map[string]any) (ResolvedDomainMTLS, error) {
	var m ResolvedDomainMTLS

	v, ok := raw["mtls"]
	if !ok {
		return m, nil
	}
	mtlsMap, ok := v.(map[string]any)
	if !ok {
		return m, fmt.Errorf("field \"mtls\" must be an object")
	}

	if enforced, ok := mtlsMap["enforced"].(bool); ok {
		m.Enforced = enforced
	}

	return m, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
// -----------------------------------------------------------------------------

// requireString extracts a non-empty string from m[key].
// Returns an error — resolution must never panic and crash the controller.
func requireString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("field %q must be a non-empty string", key)
	}
	return s, nil
}

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
