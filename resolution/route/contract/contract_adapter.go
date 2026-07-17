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
Package contract owns the proto projection for Route — a one-way mapping
from the typed resolve.ResolvedRouteSpec to the canonical proto RouteSpec
used for content-addressable hashing, audit trails, and gRPC API responses.

One-way contract:

	resolve.ResolvedRouteSpec → networkscontractv1alpha1.RouteSpec

This projection is NEVER fed back into controllers. It is the canonical
serialization of a resolved Route intent — output only.

Runtime wrapper:

	RouteSpec.Runtime is *commonv1.RouteRuntime — a common/v1 wrapper message.
	The local Runtime string is mapped to the wrapper via mapRuntime().

	Supported contract values and their canonical mappings:
	  "knative-service" or "knative"  → ROUTE_RUNTIME_KNATIVE_SERVICE
	  "kubernetes-container"           → ROUTE_RUNTIME_KUBERNETES_CONTAINER
	  "gateway-api"                    → ROUTE_RUNTIME_GATEWAY_API
	  any other value                  → ROUTE_RUNTIME_UNSPECIFIED

	Kourier is Knative's default network layer — both "knative-service" and
	"knative" map to the same proto enum value.

ServiceUnitRef:

	RouteSpec.ServiceUnitRef is *networkscontractv1alpha1.ServiceUnitRef.
	The local ResolvedServiceUnitRef.Name is projected directly.
	Nil ref produces a nil proto field — callers treat absence as invalid.

See also:
  - resolution/route/resolve  — resolution entry point and Runtime type
  - resolution/route/adapter  — interface wrapper
  - pkg/routes/               — application layer consuming ResolvedRoute
*/
package contract

import (
	commonv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	networkscontractv1alpha1 "github.com/blanketops/environments-contract/blanketops/networks/v1alpha1"

	"github.com/blanketops/environments/resolution/route/resolve"
)

// ToRouteContract projects a ResolvedRoute into the canonical proto RouteSpec
// for infrastructure consumers (hashing, audit logging, gRPC serialization).
// Returns nil, nil for a nil input — callers decide whether that is an error.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func ToRouteContract(r *resolve.ResolvedRoute) (*networkscontractv1alpha1.RouteSpec, error) {
	if r == nil || r.Spec == nil {
		return nil, nil
	}

	return &networkscontractv1alpha1.RouteSpec{
		Host:           r.Spec.Host,
		Enabled:        r.Spec.Enabled,
		Path:           r.Spec.Path,
		TlsEnabled:     r.Spec.TLSEnabled,
		Runtime:        mapRuntime(r.Spec.Runtime),
		ServiceUnitRef: mapServiceUnitRef(r.Spec.ServiceUnitRef),
	}, nil
}

// mapRuntime converts the local Runtime string to the proto *commonv1.RouteRuntime
// wrapper message. Absent or unknown values produce ROUTE_RUNTIME_UNSPECIFIED.
// The pre-validated Runtime constant values guarantee the known cases always match.
func mapRuntime(rt *resolve.Runtime) *commonv1.RouteRuntime {
	if rt == nil {
		return &commonv1.RouteRuntime{
			Runtime: commonv1.RouteRuntime_ROUTE_RUNTIME_UNSPECIFIED,
		}
	}
	switch *rt {
	case resolve.RuntimeKnativeService:
		return &commonv1.RouteRuntime{
			Runtime: commonv1.RouteRuntime_ROUTE_RUNTIME_KNATIVE_SERVICE,
		}
	case resolve.RuntimeKubernetesContainer:
		return &commonv1.RouteRuntime{
			Runtime: commonv1.RouteRuntime_ROUTE_RUNTIME_KUBERNETES_CONTAINER,
		}
	case resolve.RuntimeGatewayAPI:
		return &commonv1.RouteRuntime{
			Runtime: commonv1.RouteRuntime_ROUTE_RUNTIME_GATEWAY_API,
		}
	default:
		return &commonv1.RouteRuntime{
			Runtime: commonv1.RouteRuntime_ROUTE_RUNTIME_UNSPECIFIED,
		}
	}
}

// mapServiceUnitRef projects the local ResolvedServiceUnitRef to the proto
// ServiceUnitRef wrapper. Returns nil for a nil input — the proto field is
// optional at the wire level but treated as required by the controller.
func mapServiceUnitRef(ref *resolve.ResolvedServiceUnitRef) *networkscontractv1alpha1.ServiceUnitRef {
	if ref == nil {
		return nil
	}
	return &networkscontractv1alpha1.ServiceUnitRef{
		Name: ref.Name,
	}
}
