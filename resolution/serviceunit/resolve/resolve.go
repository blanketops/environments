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
Package resolve implements resolution for the ServiceUnit CR.

The ServiceUnit CR stores its spec as a raw JSON contract (spec.contract).
ResolveServiceUnit decodes this raw contract into a fully typed
ResolvedServiceUnit — the authoritative runtime representation consumed by
all downstream domain and application logic. This mirrors the decode style
used by every other resolver in this repo (resolution/build, resolution/route,
resolution/deployment, resolution/domain): a manual json.Unmarshal into
map[string]any, not protojson into the generated proto struct.

Resolution is strict: required fields that are missing or malformed return
errors immediately. All failures surface as errors — resolution never panics.

Image resolution for BUILD-type units (reading the referenced Build CR's
produced image) is explicitly out of scope here — every resolver in this
repo is pure/stateless, taking an already-fetched CR object with no cross-CR
reads. Image stays empty for BUILD type until that's designed separately;
downstream (intent/deployment/serviceunit.go) already guards this with
ErrBuildNotReady.
*/
package resolve

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedServiceUnit is the single runtime representation of a ServiceUnit CR.
// All downstream domain and application logic MUST use this type.
type ResolvedServiceUnit struct {
	ServiceUnit *environmentv1alpha1.ServiceUnit
	Spec        *ResolvedServiceUnitSpec
}

// ResolvedServiceUnitSpec is the decoded and validated ServiceUnit spec.
//
// Type stores the nested enum value (commoncontractv1.ServiceUnitType_ServiceUnitType)
// rather than the proto wrapper message — this allows direct comparison in
// type switches without dereferencing a pointer.
type ResolvedServiceUnitSpec struct {
	// Type discriminates between STATIC, BUILD, and SUPPLYCHAIN source modes.
	Type commoncontractv1.ServiceUnitType_ServiceUnitType
	// Image is the fully qualified static image reference.
	// Present only when Type is SERVICE_UNIT_TYPE_STATIC. Left empty for
	// BUILD type — see package doc for why that injection is out of scope here.
	Image string
	// BuildRef is the cross-CR reference to the source Build.
	// Present only when Type is SERVICE_UNIT_TYPE_BUILD.
	BuildRef      *ResolvedBuildRef
	ContainerPort int32
	Size          int32
	AppType       string
	StackType     string
	// RouteRef is the optional cross-CR reference to the Route CR exposing
	// this unit. Nil when the contract declares no route.
	RouteRef *ResolvedRouteRef
}

// ResolvedBuildRef is the resolved Build CR reference for BUILD-type service units.
type ResolvedBuildRef struct {
	Name      string
	Namespace string
}

// ResolvedRouteRef is the resolved Route CR reference for this service unit.
type ResolvedRouteRef struct {
	Name string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveServiceUnit decodes the raw JSON contract from the ServiceUnit CR
// spec into a ResolvedServiceUnit. Returns an error if the CR is nil, the
// contract is absent, or the type-specific required fields are missing.
func ResolveServiceUnit(su *environmentv1alpha1.ServiceUnit) (*ResolvedServiceUnit, error) {
	if su == nil {
		return nil, fmt.Errorf("serviceunit is nil")
	}

	if len(su.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(su.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Type (REQUIRED).
	// ------------------------------------------------
	unitType, err := resolveServiceUnitType(raw["type"])
	if err != nil {
		return nil, err
	}

	spec := &ResolvedServiceUnitSpec{
		Type:          unitType,
		ContainerPort: optionalInt32(raw, "containerPort"),
		Size:          optionalInt32(raw, "size"),
		AppType:       optionalString(raw, "appType"),
		StackType:     optionalString(raw, "stackType"),
	}

	// ------------------------------------------------
	// Type-specific field validation and extraction.
	// ------------------------------------------------
	switch unitType {
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		image, err := mustString(raw, "image")
		if err != nil {
			return nil, fmt.Errorf("image: %w", err)
		}
		spec.Image = image

	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		brRaw, ok := raw["buildRef"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("buildRef is required when type is BUILD")
		}
		name, err := mustString(brRaw, "name")
		if err != nil {
			return nil, fmt.Errorf("buildRef.name: %w", err)
		}
		spec.BuildRef = &ResolvedBuildRef{
			Name:      name,
			Namespace: optionalString(brRaw, "namespace"),
		}

	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_SUPPLYCHAIN:
		// No additional required fields defined by the contract yet.

	default:
		return nil, fmt.Errorf("unsupported serviceunit.type %q", unitType.String())
	}

	// ------------------------------------------------
	// Route (OPTIONAL).
	// ------------------------------------------------
	if routeRaw, ok := raw["route"].(map[string]any); ok {
		name, err := mustString(routeRaw, "name")
		if err != nil {
			return nil, fmt.Errorf("route.name: %w", err)
		}
		spec.RouteRef = &ResolvedRouteRef{Name: name}
	}

	return &ResolvedServiceUnit{
		ServiceUnit: su,
		Spec:        spec,
	}, nil
}

// resolveServiceUnitType maps the raw contract value to the nested enum.
// Accepts short lowercase strings ("static", "build", "supplychain") as
// well as numeric contract encoding, mirroring resolution/deployment's
// resolveDeploymentRuntime.
func resolveServiceUnitType(raw any) (commoncontractv1.ServiceUnitType_ServiceUnitType, error) {
	switch v := raw.(type) {
	case string:
		switch v {
		case "static", "STATIC":
			return commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC, nil
		case "build", "BUILD":
			return commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD, nil
		case "supplychain", "SUPPLYCHAIN":
			return commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_SUPPLYCHAIN, nil
		default:
			return 0, fmt.Errorf("unsupported serviceunit.type %q", v)
		}
	case float64:
		return commoncontractv1.ServiceUnitType_ServiceUnitType(v), nil
	default:
		return 0, fmt.Errorf("serviceunit.type is required")
	}
}

// -----------------------------------------------------------------------------
// Extraction helpers — no panics
// -----------------------------------------------------------------------------

func mustString(m map[string]any, key string) (string, error) {
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

func optionalInt32(m map[string]any, key string) int32 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int32(n)
		case int:
			return int32(n)
		}
	}
	return 0
}
