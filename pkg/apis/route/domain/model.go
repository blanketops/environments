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
This file owns the Route domain aggregate and the Runtime typed constant set.

Route is the authoritative in-memory representation of a Route CR after
resolution. All downstream application and provider logic operates exclusively
on this type — no raw strings, no re-reads of the Kubernetes CR.

ServiceRef carries the name of the K8s Service or Knative Service this route
exposes. It is derived by the Mapper directly from spec.ServiceUnitRef.Name
resolved from the contract — no label lookup, no API server read, no status
lookup. Platform convention enforces: ksvc name == ServiceUnit name.

See also:
  - pkg/route/domain/state.go    — Phase type and constants
  - pkg/route/domain/result.go   — RouteResult returned by the provider
  - pkg/route/application/mapper.go — constructs Route from ResolvedRouteSpec
  - pkg/route/api/provider.go    — Provider interface consuming Route
*/
package domain

// Route is the authoritative domain aggregate for the Route CR.
// Constructed once by the Mapper; consumed by the application service and
// provider implementations. Never mutated after construction.
type Route struct {
	// Name is the Route CR name.
	Name string

	// Namespace is the Route CR namespace (tenant/environment namespace).
	Namespace string

	// Host is the FQDN this route serves (e.g. api.dev.blanketops.online).
	Host string

	// Enabled controls whether the route is active.
	// When false the provider removes the runtime resource but retains the CR.
	Enabled bool

	// Path is the HTTP path prefix to match (e.g. /, /v1, /api).
	Path string

	// TLSEnabled controls whether TLS termination is enforced at the runtime.
	TLSEnabled bool

	// Runtime is the serving backend that materializes this route.
	// BackendSelector branches on this to select the correct Provider.
	Runtime Runtime

	// ServiceRef is the name of the K8s Service or Knative Service backing
	// this route. Derived by Mapper from spec.ServiceUnitRef.Name.
	// Convention: ksvc name == ServiceUnit name — no label, no lookup.
	// Both KnativeProvider and IngressProvider guard emptiness with
	// ErrServiceRefEmpty at Ensure time as a final safety net.
	ServiceRef string
}

// Runtime identifies the serving backend that materializes a Route.
// Mirrors blanketops.common.v1.RouteRuntime in the proto contract.
type Runtime string

const (
	// RuntimeKnativeService materializes the route as a Knative DomainMapping
	// served via Kourier. The DomainMapping binds the host to a Knative Service.
	// Implemented by KnativeProvider.
	RuntimeKnativeService Runtime = "knative-service"

	// RuntimeKubernetesContainer materializes the route as a standard Kubernetes
	// Ingress resource via nginx. For non-Knative workloads.
	// Implemented by IngressProvider.
	RuntimeKubernetesContainer Runtime = "kubernetes-container"

	// RuntimeGatewayAPI materializes the route as a Gateway API HTTPRoute.
	// Future — not yet implemented. BackendSelector returns ErrRuntimeUnknown.
	RuntimeGatewayAPI Runtime = "gateway-api"
)
