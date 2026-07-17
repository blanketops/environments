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
This file owns Mapper — the translation layer between the resolved
ServiceUnit contract and the domain ServiceUnit aggregate consumed by the
application service.

The Mapper enforces the resolution contract: it panics on fields the
resolver guarantees to be present or valid (a recognised Type) so that
resolver bugs surface loudly rather than silently producing a mistyped
aggregate. This mirrors pkg/apis/route/application/mapper.go's convention.
*/
package application

import (
	"fmt"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"

	"github.com/blanketops/environments/pkg/apis/serviceunit/domain"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

// Mapper translates a ResolvedServiceUnit into a domain.ServiceUnit.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved ServiceUnit into a domain
// ServiceUnit for consumption by the application service.
//
// Panics on resolver invariant violations (unrecognised Type) — these
// indicate a resolver bug, not a user error, and must not be silently
// swallowed. All other fields are mapped verbatim.
func (Mapper) MapResolvedToDomain(rsu *serviceunitResolution.ResolvedServiceUnit) domain.ServiceUnit {
	spec := rsu.Spec

	unitType, err := toDomainType(spec.Type)
	if err != nil {
		panic(fmt.Sprintf("resolved serviceunit %q: %v (resolver bug)", rsu.ServiceUnit.Name, err))
	}

	su := domain.ServiceUnit{
		Name:          rsu.ServiceUnit.Name,
		Namespace:     rsu.ServiceUnit.Namespace,
		Type:          unitType,
		Image:         spec.Image,
		ContainerPort: spec.ContainerPort,
		Size:          spec.Size,
		AppType:       spec.AppType,
		StackType:     spec.StackType,
	}

	if spec.BuildRef != nil {
		su.BuildRef = &domain.BuildRef{
			Name:      spec.BuildRef.Name,
			Namespace: spec.BuildRef.Namespace,
		}
	}

	if spec.RouteRef != nil {
		su.RouteRef = &domain.RouteRef{Name: spec.RouteRef.Name}
	}

	return su
}

// toDomainType converts the contract-layer enum to the domain-owned Type.
func toDomainType(t commoncontractv1.ServiceUnitType_ServiceUnitType) (domain.Type, error) {
	switch t {
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC:
		return domain.TypeStatic, nil
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD:
		return domain.TypeBuild, nil
	case commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_SUPPLYCHAIN:
		return domain.TypeSupplyChain, nil
	default:
		return "", fmt.Errorf("unrecognised type %v", t)
	}
}
