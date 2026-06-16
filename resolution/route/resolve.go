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
Package Route implements resolution for the Route CR.

This file owns the Adapter — a thin struct wrapper around the package-level
ResolveDeployment function. The Adapter exists to satisfy interface-based injection
points in the application layer where a concrete resolution dependency must be
passed as a value rather than called as a free function.

Current deps are intentionally empty — client and logger will be added here
when resolution requires cross-CR reads or observability hooks. Until then,
resolution is pure and stateless so no deps are needed.
*/
package route

import (
	networksv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/networks/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
//
// ResolvedRoute is the single runtime representation of a Route CR.
// All downstream domain and application logic MUST use this type.
// Never re-read from the raw CR spec after resolution.
// -----------------------------------------------------------------------------

// ResolvedRoute pairs the original Kubernetes Route object with its fully
// decoded and validated spec.
type ResolvedRoute struct {
	Route *networksv1alpha1.Route
	Spec  *ResolvedRouteSpec
}

// ResolvedRouteSpec is the decoded and validated Build spec.
type ResolvedRouteSpec struct {
	// Image is the fully qualified target image reference (registry/repo:tag).
	Host string

	Path string

	TLSEnabled bool

	// ServiceAccount is optional — omitted when the Build uses the default
	// Shipwright service account.
	//Runtime *Runtime
}
