package application

import (
	"fmt"

	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/domain"
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

	// if len(spec.ServiceUnits) == 0 {
	// 	panic(fmt.Sprintf(
	// 		"resolved deployment %q has no service units (resolver bug)",
	// 		rd.Deployment.Name,
	// 	))
	// }

	// if spec.Runtime == 0 {
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
