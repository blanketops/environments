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

package application

import (
	"fmt"

	deploymentResolution "github.com/BlanketOps/environments/resolution/deployment"
	"github.com/BlanketOps/blanketops-environments/pkg/apis/deployment/domain"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved deployment into a pure domain DeploymentSpec.
//
// CONTRACT:
// - Resolver guarantees presence of mandatory fields
// - Nil values indicate a resolver bug and MUST crash loudly
// - Optional fields must be preserved verbatim
// - Mapper must not invent defaults or hide intent
func (Mapper) MapResolvedToDomain(
	rd *deploymentResolution.ResolvedDeployment,
) domain.DeploymentSpec {

	if rd == nil || rd.Spec == nil {
		panic("nil ResolvedDeployment (resolver bug)")
	}

	spec := rd.Spec

	// ---------------------------------------------------------------------
	// INVARIANTS (resolver-owned guarantees)
	// ---------------------------------------------------------------------

	if len(spec.ServiceUnits) == 0 {
		panic(fmt.Sprintf(
			"resolved deployment %q has no service units (resolver bug)",
			rd.Deployment.Name,
		))
	}

	// if spec.Runtime == "" {
	// 	panic(fmt.Sprintf(
	// 		"resolved deployment %q has empty runtime (resolver bug)",
	// 		rd.Deployment.Name,
	// 	))
	// }

	// if spec.ReconciliationStrategy == 0 {
	// 	panic(fmt.Sprintf(
	// 		"resolved deployment %q has empty reconciliationStrategy (resolver bug)",
	// 		rd.Deployment.Name,
	// 	))
	// }

	// ---------------------------------------------------------------------
	// ManifestsRepo (OPTIONAL – preserved exactly)
	// ---------------------------------------------------------------------

	var repo *domain.ManifestsRepo
	if spec.ManifestsRepo != nil {

		if spec.ManifestsRepo.URL == "" {
			panic(fmt.Sprintf(
				"resolved deployment %q has empty manifestsRepo.url (resolver bug)",
				rd.Deployment.Name,
			))
		}

		repo = &domain.ManifestsRepo{
			URL:         spec.ManifestsRepo.URL,
			Ref:         spec.ManifestsRepo.Ref,
			CloneSecret: spec.ManifestsRepo.CloneSecret,
			Strategy:    spec.ManifestsRepo.Strategy,
			Path:        spec.ManifestsRepo.Path,
		}
	}

	// ---------------------------------------------------------------------
	// Domain boundary mapping (BORING ON PURPOSE)
	// ---------------------------------------------------------------------

	return domain.DeploymentSpec{
		Name: rd.Deployment.Name,

		ServiceUnits: spec.ServiceUnits,

		Runtime: domain.Runtime(spec.Runtime),

		ImageAutomation: spec.ImageAutomation,

		ManifestsRepo: repo,

		ReconciliationStrategy: domain.ReconciliationStrategy(spec.ReconciliationStrategy),
	}
}
