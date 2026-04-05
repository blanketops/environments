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

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToServiceUnitContract converts a resolved runtime ServiceUnit spec into a
// canonical ServiceUnit CONTRACT spec.
//
// ⚠️ ONE-WAY ADAPTER
// - For hashing, diffing, storage, and legacy consumers only
// - Controllers must NEVER consume the returned value
func (s *ResolvedServiceUnitSpec) ToServiceUnitContract() *contractv1.ServiceUnitSpec {
	// Absolute guard: no spec, no contract
	if s == nil {
		return nil
	}

	out := &contractv1.ServiceUnitSpec{
		Type:          s.Type,
		ContainerPort: s.ContainerPort,
		Size:          s.Size,
		AppType:       s.AppType,
		StackType:     s.StackType,
	}

	// ------------------------------------------------
	// Type-specific fields
	// ------------------------------------------------
	switch s.Type {

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		// Resolver guarantees Image is present
		out.Image = s.Image

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		// Resolver guarantees BuildRef is present
		out.BuildRef = &contractv1.BuildReference{
			Name:      s.BuildRef.Name,
			Namespace: s.BuildRef.Namespace,
		}
	}

	return out
}
