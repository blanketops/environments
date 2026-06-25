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
This file owns the Phase type and its constants for the Route domain.

Phase mirrors blanketops.common.v1.RoutePhase in the proto contract. The
status writer (pkg/routes/application/status.go) maps Phase values to the
conditions on the Route CR status subresource.

See also:
  - pkg/routes/domain/result.go — RouteResult carries Phase as its outcome
  - pkg/routes/application/status.go — writes Phase to the CR status
*/
package domain

// Phase represents the current lifecycle phase of a Route CR.
// Set by the status writer after each provider Ensure call.
type Phase string

const (
	// PhasePending indicates the controller has accepted the Route CR but
	// the DomainMapping is not yet ready (backend not serving, TLS pending).
	PhasePending Phase = "Pending"

	// PhaseReady indicates the DomainMapping is active and the host is serving
	// traffic. TLS is provisioned when TLSEnabled is true.
	PhaseReady Phase = "Ready"

	// PhaseDegraded indicates the route is serving but unhealthy — for example,
	// the TLS cert is expiring or the backend is flapping.
	PhaseDegraded Phase = "Degraded"

	// PhaseFailed indicates the route could not be materialized.
	// Check RouteResult.Message and the CR conditions for detail.
	PhaseFailed Phase = "Failed"
)
