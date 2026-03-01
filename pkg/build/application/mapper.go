package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	bldResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved build into a pure domain BuildSpec.
//
// CONTRACT:
// - Resolver guarantees presence of mandatory fields
// - Nil values indicate a resolver bug and MUST crash loudly
// - Optional fields must be preserved verbatim
// - Mapper must not invent defaults or hide intent
func (Mapper) MapResolvedToDomain(rb *bldResolution.ResolvedBuild) domain.BuildSpec {
	spec := rb.Spec

	// ---------------------------------------------------------------------
	// INVARIANTS (resolver-owned guarantees)
	// ---------------------------------------------------------------------
	if spec.Source.URL == "" {
		panic(fmt.Sprintf(
			"resolved build %q has empty Source.Url (resolver bug)",
			rb.Build.Name,
		))
	}

	if spec.Strategy.Name == "" {
		panic(fmt.Sprintf(
			"resolved build %q has empty Strategy.Name (resolver bug)",
			rb.Build.Name,
		))
	}

	// ---------------------------------------------------------------------
	// ServiceAccount (OPTIONAL – preserved exactly)
	// ---------------------------------------------------------------------
	var saName string
	var saSecret string
	if spec.ServiceAccount != nil {
		saName = spec.ServiceAccount.Name
		saSecret = spec.ServiceAccount.Secret
	}

	// ---------------------------------------------------------------------
	// Domain boundary mapping
	// ---------------------------------------------------------------------
	return domain.BuildSpec{
		// Source
		SourceURL:   spec.Source.URL,
		ContextDir:  spec.Source.ContextDir,
		Revision:    spec.Source.Revision,
		CloneSecret: spec.Source.CloneSecret, // OPTIONAL but preserved

		// Strategy
		StrategyName: spec.Strategy.Name,
		StrategyKind: "ClusterBuildStrategy",

		// Output
		Image: spec.Image,

		// ServiceAccount (optional)
		ServiceAccountName:   saName,
		ServiceAccountSecret: saSecret,
	}
}
