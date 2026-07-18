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

package intent

import (
	"fmt"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	serviceunit "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

type ServiceUnitIntent struct {
	Name   string
	Image  string
	Port   int32
	Size   int32
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
		Name: su.ServiceUnit.Name,
		Port: spec.ContainerPort,
		Size: spec.Size,
	}

	// ------------------------------------------------
	// Image (STATIC / BUILD already resolved upstream)
	//
	// spec.Type is commoncontractv1.ServiceUnitType_ServiceUnitType —
	// the nested enum value stored by the resolver, not the wrapper message.
	// ------------------------------------------------
	switch spec.Type {
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		if spec.Image == "" {
			return nil, ErrInvalidServiceUnit(
				intent.Name,
				"static serviceunit has empty image (resolver bug)",
			)
		}
		intent.Image = spec.Image

	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		// BUILD image MUST be injected during resolution before reaching intent.
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
	//
	// Routes must be resolved BEFORE the intent layer.
	// Intent does not inspect CR route specs.
	// ------------------------------------------------

	return intent, nil
}
