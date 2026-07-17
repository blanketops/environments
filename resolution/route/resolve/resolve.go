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
Package resolve implements resolution for the Route CR.

The Route CR stores its spec as a raw JSON contract (spec.contract).
ResolveRoute decodes this into a fully typed ResolvedRoute — the authoritative
runtime representation consumed by all downstream domain and application logic.

Route declares the intent to expose a ServiceUnit at a host and path. The
controller materializes the route through the selected runtime (Kubernetes
Ingress, Knative DomainMapping, or Gateway API HTTPRoute). Owned by an
Environment CR via ownerReference (cascade delete).

Workload resolution:

	Route.spec.serviceUnitRef names the ServiceUnit in the same namespace.
	The controller derives the ksvc/K8s service name by convention:
	  ksvc name == ServiceUnit name
	No ServiceUnit status lookup is required during resolution.

All failures surface as errors. Resolution never panics.
*/
package resolve

import (
	"encoding/json"
	"fmt"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime type
// -----------------------------------------------------------------------------

// Runtime identifies the serving backend that materializes this Route.
// Corresponds to blanketops.common.v1.RouteRuntime in the proto contract.
type Runtime string

const (
	// RuntimeKnativeService materializes the route as a Knative DomainMapping
	// via Kourier. Covers both "knative-service" and "knative" contract values.
	RuntimeKnativeService Runtime = "knative-service"

	// RuntimeKubernetesContainer materializes the route as a standard Kubernetes
	// Ingress resource. Covers "kubernetes-container" contract values.
	RuntimeKubernetesContainer Runtime = "kubernetes-container"

	// RuntimeGatewayAPI materializes the route as a Gateway API HTTPRoute.
	// Future — not yet supported by any backend provider.
	RuntimeGatewayAPI Runtime = "gateway-api"
)

// -----------------------------------------------------------------------------
// Resolved types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedRoute pairs the original Kubernetes Route object with its fully
// decoded and validated spec.
type ResolvedRoute struct {
	Route *networksv1alpha1.Route
	Spec  *ResolvedRouteSpec
}

// ResolvedRouteSpec is the decoded and validated Route spec.
// Fields mirror the proto RouteSpec — consumers must not re-read from the raw CR.
type ResolvedRouteSpec struct {
	// Host is the fully qualified domain name to serve.
	// e.g. app.dev.blanketops.online
	Host string

	// Enabled controls whether the route is active. Defaults to true.
	// Disabled routes are removed from the runtime but the CR is retained.
	Enabled bool

	// Path is the HTTP path prefix to match.
	// e.g. /, /v1, /api. Defaults to "/" when absent.
	Path string

	// TLSEnabled controls whether TLS termination is enforced.
	// Certificate provisioning is delegated to cert-manager via the platform
	// issuer (DNS01 wildcard or namespaced HTTP01). Defaults to true.
	TLSEnabled bool

	// Runtime is the serving backend responsible for materializing this route.
	// Mandatory — backend selection in pkg/routes/application/ branches on this.
	Runtime *Runtime

	// ServiceUnitRef identifies the ServiceUnit this Route exposes.
	// Mandatory — a Route without a workload binding is invalid.
	// The controller derives the ksvc/K8s service name by convention:
	//   ksvc name == ServiceUnitRef.Name
	// No ServiceUnit status lookup is required.
	ServiceUnitRef *ResolvedServiceUnitRef
}

// ResolvedServiceUnitRef is the decoded reference to a ServiceUnit CR.
// Mirrors proto ServiceUnitRef — name only; namespace is always this Route's namespace.
type ResolvedServiceUnitRef struct {
	// Name is the ServiceUnit CR name in the same namespace as this Route.
	Name string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveRoute decodes and validates the raw JSON contract from the Route CR
// spec into a ResolvedRoute. Returns an error if the CR is nil, the contract
// is absent, or any required field is missing or malformed.
func ResolveRoute(route *networksv1alpha1.Route) (*ResolvedRoute, error) {
	if route == nil {
		return nil, fmt.Errorf("route is nil")
	}
	if len(route.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(route.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// --------------------------------------------------------
	// Required fields.
	// --------------------------------------------------------

	host, err := requireString(raw, "host")
	if err != nil {
		return nil, err
	}

	runtimeStr, err := requireString(raw, "runtime")
	if err != nil {
		return nil, err
	}
	rt := Runtime(runtimeStr)

	// serviceUnitRef is required — Route without a workload binding is invalid.
	serviceUnitRef, err := requireServiceUnitRef(raw)
	if err != nil {
		return nil, err
	}

	// --------------------------------------------------------
	// Optional fields — platform defaults apply when absent.
	// --------------------------------------------------------

	// enabled: absent or true means the route is active.
	// Explicit false removes the materialized resource while retaining the CR.
	enabled := true
	if v, ok := raw["enabled"].(bool); ok {
		enabled = v
	}

	// path: absent defaults to "/" — match all paths under the host.
	path := optionalString(raw, "path")
	if path == "" {
		path = "/"
	}

	// tlsEnabled: absent defaults to true — plain HTTP requires explicit false.
	tlsEnabled := true
	if v, ok := raw["tlsEnabled"].(bool); ok {
		tlsEnabled = v
	}

	return &ResolvedRoute{
		Route: route,
		Spec: &ResolvedRouteSpec{
			Host:           host,
			Enabled:        enabled,
			Path:           path,
			TLSEnabled:     tlsEnabled,
			Runtime:        &rt,
			ServiceUnitRef: serviceUnitRef,
		},
	}, nil
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

// requireServiceUnitRef extracts and validates the serviceUnitRef nested object.
// Expected contract shape: {"serviceUnitRef": {"name": "my-service-unit"}}
// Returns an error if the field is absent, not an object, or name is empty.
func requireServiceUnitRef(m map[string]any) (*ResolvedServiceUnitRef, error) {
	v, ok := m["serviceUnitRef"]
	if !ok {
		return nil, fmt.Errorf("missing required field \"serviceUnitRef\"")
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field \"serviceUnitRef\" must be an object")
	}
	name, ok := obj["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("field \"serviceUnitRef.name\" must be a non-empty string")
	}
	return &ResolvedServiceUnitRef{Name: name}, nil
}
