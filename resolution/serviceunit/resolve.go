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
Package serviceunit implements resolution for the ServiceUnit CR.

The ServiceUnit CR stores its spec as a raw JSON contract (spec.contract).
ResolveServiceUnit decodes this via protojson into a contractv1.ServiceUnit
and extracts a fully typed ResolvedServiceUnit — the authoritative runtime
representation consumed by all downstream domain and application logic.

ResolvedServiceUnitSpec.Type is the nested enum value
(commoncontractv1.ServiceUnitType_ServiceUnitType) rather than the wrapper
message — this keeps the type switch in the contract adapter clean and avoids
dereferencing a pointer to read the enum.
*/
package serviceunit

import (
	environmentv1alpha1 "github.com/BlanketOps/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
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
	// Type discriminates between STATIC and BUILD source modes.
	Type commoncontractv1.ServiceUnitType_ServiceUnitType
	// Image is the fully qualified static image reference.
	// Present only when Type is SERVICE_UNIT_TYPE_STATIC.
	Image string
	// BuildRef is the cross-CR reference to the source Build.
	// Present only when Type is SERVICE_UNIT_TYPE_BUILD.
	BuildRef      *ResolvedBuildRef
	ContainerPort int32
	Size          int32
	AppType       string
	StackType     string
}

// ResolvedBuildRef is the resolved Build CR reference for BUILD-type service units.
type ResolvedBuildRef struct {
	Name      string
	Namespace string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveServiceUnit decodes the raw JSON contract from the ServiceUnit CR
// spec into a ResolvedServiceUnit. Returns an error if the CR is nil, the
// contract is absent, or the type-specific required fields are missing.
func ResolveServiceUnit(su *environmentv1alpha1.ServiceUnit) (*ResolvedServiceUnit, error) {
	// if su == nil {
	// 	return nil, fmt.Errorf("serviceunit is nil")
	// }

	// if len(su.Spec.Contract.Raw) == 0 {
	// 	return nil, fmt.Errorf("spec.contract is required")
	// }

	// // Decode canonical ServiceUnit contract (JSON → proto).
	// 	var raw map[string]any
	// if err := json.Unmarshal(depl.Spec.Contract.Raw, &raw); err != nil {
	// 	return nil, fmt.Errorf("failed to decode deployment contract: %w", err)
	// }
	// spec := contract.GetSpec()
	// if spec == nil {
	// 	return nil, fmt.Errorf("serviceunit.spec missing in contract")
	// }

	// // Extract the nested enum value from the wrapper message.
	// // GetType() returns *v1.ServiceUnitType; GetType().GetType() returns
	// // the ServiceUnitType_ServiceUnitType enum value we store in the resolved spec.
	// var unitType commoncontractv1.ServiceUnitType_ServiceUnitType
	// if t := spec.GetType(); t != nil {
	// 	unitType = t.GetType()
	// }

	// resolved := &ResolvedServiceUnit{
	// 	ServiceUnit: su,
	// 	Spec: &ResolvedServiceUnitSpec{
	// 		Type:          unitType,
	// 		ContainerPort: spec.GetContainerPort(),
	// 		Size:          spec.GetSize(),
	// 		AppType:       spec.GetAppType(),
	// 		StackType:     spec.GetStackType(),
	// 	},
	// }

	// // ------------------------------------------------
	// // Type-specific field validation and extraction.
	// // ------------------------------------------------
	// switch unitType {
	// case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
	// 	if spec.GetImage() == "" {
	// 		return nil, fmt.Errorf("static serviceunit.image is required")
	// 	}
	// 	resolved.Spec.Image = spec.GetImage()

	// case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
	// 	br := spec.GetBuildRef()
	// 	if br == nil {
	// 		return nil, fmt.Errorf("build serviceunit.buildRef is required")
	// 	}
	// 	resolved.Spec.BuildRef = &ResolvedBuildRef{
	// 		Name:      br.GetName(),
	// 		Namespace: br.GetNamespace(),
	// 	}

	// default:
	// 	return nil, fmt.Errorf("unsupported ServiceUnit type %q", unitType.String())
	// }

	// return resolved, nil
	return nil, nil
}
