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

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedServiceUnitSpec) to the protobuf contract type
(environmentscontractv1alpha1.ServiceUnitSpec).

ResolvedServiceUnitSpec.Type is commoncontractv1.ServiceUnitType_ServiceUnitType
(the nested enum value). The contract field ServiceUnitSpec.Type is
*v1.ServiceUnitType (the wrapper message). toServiceUnitTypeWrapper bridges
the two — the type switch operates on the enum value, not the wrapper.

⚠️ ONE-WAY adapter. Controllers must NEVER consume the returned value.
*/
package serviceunit

import (
	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
	environmentscontractv1alpha1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

// ToServiceUnitContract projects the resolved runtime ServiceUnit spec into a
// protobuf environmentscontractv1alpha1.ServiceUnitSpec for infrastructure
// consumers (hashing, diffing, storage, audit).
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedServiceUnitSpec) ToServiceUnitContract() *environmentscontractv1alpha1.ServiceUnitSpec {
	if s == nil {
		return nil
	}

	out := &environmentscontractv1alpha1.ServiceUnitSpec{
		Type:          toServiceUnitTypeWrapper(s.Type),
		ContainerPort: s.ContainerPort,
		Size:          s.Size,
		AppType:       s.AppType,
		StackType:     s.StackType,
	}

	switch s.Type {
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		// Resolver guarantees Image is present for STATIC type.
		out.Image = s.Image

	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		// Resolver guarantees BuildRef is non-nil for BUILD type.
		if s.BuildRef != nil {
			out.BuildRef = &environmentscontractv1alpha1.BuildReference{
				Name:      s.BuildRef.Name,
				Namespace: s.BuildRef.Namespace,
			}
		}
	}

	return out
}

// toServiceUnitTypeWrapper wraps the nested enum value in the
// *v1.ServiceUnitType wrapper message expected by the generated proto.
func toServiceUnitTypeWrapper(t commoncontractv1.ServiceUnitType_ServiceUnitType) *commoncontractv1.ServiceUnitType {
	return &commoncontractv1.ServiceUnitType{Type: t}
}
