package intent

import (
	"fmt"

	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/serviceunit"
)

type ServiceUnitIntent struct {
	Name string

	Image string
	Port  int32
	Size  int32

	Routes []RouteIntent

	// Filled after execution
	Workload WorkloadIntent
}

func ResolveServiceUnitIntent(
	su *serviceunit.ResolvedServiceUnit,
) (*ServiceUnitIntent, error) {

	if su == nil || su.Spec == nil {
		return nil, fmt.Errorf("resolved serviceunit is nil")
	}

	spec := su.Spec

	intent := &ServiceUnitIntent{
		Name: su.ServiceUnit.Name, // ✅ metadata only

		Port: spec.ContainerPort,
		Size: spec.Size,
	}

	// ------------------------------------------------
	// Image (STATIC / BUILD already resolved upstream)
	// ------------------------------------------------
	switch spec.Type {

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		if spec.Image == "" {
			return nil, ErrInvalidServiceUnit(
				intent.Name,
				"static serviceunit has empty image (resolver bug)",
			)
		}
		intent.Image = spec.Image

	case contractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		// BUILD image MUST be injected during resolution
		if spec.Image == "" {
			return nil, ErrBuildNotReady(intent.Name)
		}
		intent.Image = spec.Image

	default:
		return nil, ErrInvalidServiceUnit(
			intent.Name,
			"unsupported serviceunit type",
		)
	}

	// ------------------------------------------------
	// Routes
	// ------------------------------------------------
	// NOTE:
	// Routes must be resolved BEFORE intent layer.
	// Intent does not inspect CR route specs.

	return intent, nil
}
