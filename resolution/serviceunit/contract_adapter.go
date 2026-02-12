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
