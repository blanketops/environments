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
This file owns the ServiceUnit domain aggregate and the Type typed constant
set.

ServiceUnit is the authoritative in-memory representation of a ServiceUnit CR
after resolution. All downstream application logic operates exclusively on
this type — no raw strings, no re-reads of the Kubernetes CR.

Unlike Build, Route, or the Domain CR, a ServiceUnit does not materialize
anything onto infrastructure itself — per the contract, it is "materialized
onto a runtime by a Deployment CR." Its own reconciliation only resolves and
reports state (see state.go, result.go); there is no BackendSelector or
Provider layer here.

See also:
  - pkg/apis/serviceunit/domain/state.go      — Phase type and constants
  - pkg/apis/serviceunit/domain/result.go     — Result returned by Reconcile
  - pkg/apis/serviceunit/application/mapper.go — constructs ServiceUnit from ResolvedServiceUnit
*/
package domain

// ServiceUnit is the authoritative domain aggregate for the ServiceUnit CR.
// Constructed once by the Mapper; consumed by the application service.
// Never mutated after construction.
type ServiceUnit struct {
	// Name is the ServiceUnit CR name.
	Name string

	// Namespace is the ServiceUnit CR namespace.
	Namespace string

	// Type discriminates how this unit's image is sourced.
	Type Type

	// Image is the fully qualified image reference. Populated for STATIC
	// units; empty for BUILD units until Build-image injection is designed
	// (see resolution/serviceunit package doc).
	Image string

	// ContainerPort is the port the workload listens on.
	ContainerPort int32

	// Size is the desired replica count.
	Size int32

	// AppType is an application category hint (e.g. web, worker, job).
	AppType string

	// StackType is a language/stack hint (e.g. nodejs, python, wasm).
	StackType string

	// BuildRef is the cross-CR reference to the source Build.
	// Non-nil only when Type is TypeBuild.
	BuildRef *BuildRef

	// RouteRef is the optional cross-CR reference to the Route exposing
	// this unit. Nil when the contract declares no route.
	RouteRef *RouteRef
}

// BuildRef identifies the source Build CR for a BUILD-type ServiceUnit.
type BuildRef struct {
	Name      string
	Namespace string
}

// RouteRef identifies the Route CR exposing a ServiceUnit.
type RouteRef struct {
	Name string
}

// Type discriminates how a ServiceUnit's image is sourced.
// Mirrors blanketops.common.v1.ServiceUnitType in the proto contract.
type Type string

const (
	// TypeStatic pins an image reference directly on the ServiceUnit.
	TypeStatic Type = "static"

	// TypeBuild resolves the image from a Build CR's output.
	TypeBuild Type = "build"

	// TypeSupplyChain sources the image via a supply-chain pipeline.
	// Declared by the contract; no additional required fields yet.
	TypeSupplyChain Type = "supplychain"
)
