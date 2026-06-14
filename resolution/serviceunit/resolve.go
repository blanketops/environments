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

package serviceunit

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

// ResolvedServiceUnit is the SINGLE runtime representation of a ServiceUnit.
// Everything downstream MUST use this.
type ResolvedServiceUnit struct {
	ServiceUnit *environmentv1.ServiceUnit
	Spec        *ResolvedServiceUnitSpec
}

type ResolvedServiceUnitSpec struct {
	Type contractv1.ServiceUnitType

	// STATIC
	Image string

	// BUILD
	BuildRef *ResolvedBuildRef

	// Common
	ContainerPort int32
	Size          int32
	AppType       string
	StackType     string
}

type ResolvedBuildRef struct {
	Name      string
	Namespace string
}

func ResolveServiceUnit(
	su *environmentv1.ServiceUnit,
) (*ResolvedServiceUnit, error) {

	if su == nil {
		return nil, fmt.Errorf("serviceunit is nil")
	}

	if len(su.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	// ------------------------------------------------------------
	// Decode canonical ServiceUnit contract (JSON → proto)
	// ------------------------------------------------------------
	var contract contractv1.ServiceUnit

	if err := protojson.Unmarshal(
		su.Spec.Contract.Raw,
		&contract,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode serviceunit contract: %w",
			err,
		)
	}

	spec := contract.GetSpec()
	if spec == nil {
		return nil, fmt.Errorf("serviceunit.spec missing in contract")
	}

	// ------------------------------------------------------------
	// Resolve type-specific fields
	// ------------------------------------------------------------
	resolved := &ResolvedServiceUnit{
		ServiceUnit: su,
		Spec: &ResolvedServiceUnitSpec{
			Type:          spec.GetType(),
			ContainerPort: spec.GetContainerPort(),
			Size:          spec.GetSize(),
			AppType:       spec.GetAppType(),
			StackType:     spec.GetStackType(),
		},
	}

	switch spec.GetType() {

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		if spec.GetImage() == "" {
			return nil, fmt.Errorf("static serviceunit.image is required")
		}
		resolved.Spec.Image = spec.GetImage()

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		br := spec.GetBuildRef()
		if br == nil {
			return nil, fmt.Errorf("build serviceunit.buildRef is required")
		}
		resolved.Spec.BuildRef = &ResolvedBuildRef{
			Name:      br.GetName(),
			Namespace: br.GetNamespace(),
		}

	default:
		return nil, fmt.Errorf(
			"unsupported ServiceUnit type %q",
			spec.GetType().String(),
		)
	}

	return resolved, nil
}
