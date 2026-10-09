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
This file owns the Mapper — the translation layer between the resolved Build
contract and the domain BuildSpec consumed by the provider layer.

The Mapper is declarative: it carries over what the contract declares and
adds nothing. Fields resolution guarantees (Source.URL, Strategy.Name,
Strategy.StrategyKind) are checked again and reported as errors, never as a
panic, so a caller that bypasses resolution gets a failed build rather than a
crashed controller.

Optional fields (ServiceAccount, CloneSecret) are passed through verbatim.
*/
package application

import (
	"fmt"

	"github.com/blanketops/environments/pkg/apis/build/domain"
	bldResolution "github.com/blanketops/environments/resolution/build/resolve"
)

// Mapper translates a ResolvedBuild into a domain.BuildSpec.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved Build into a domain BuildSpec
// for consumption by the provider layer.
//
// It returns an error wrapping domain.ErrInvalidBuild when the resolved Build
// is missing something resolution should have required. All other fields are
// mapped verbatim.
//
// StrategyKind is translated from the contract's spelling to the Shipwright
// kind the providers write: a namespaced strategy is a Shipwright
// "BuildStrategy".
func (Mapper) MapResolvedToDomain(rb *bldResolution.ResolvedBuild) (domain.BuildSpec, error) {
	if rb == nil || rb.Build == nil || rb.Spec == nil {
		return domain.BuildSpec{}, fmt.Errorf("%w: no resolved build", domain.ErrInvalidBuild)
	}
	spec := rb.Spec

	if spec.Source.URL == "" {
		return domain.BuildSpec{}, fmt.Errorf("%w: build %q has no source url", domain.ErrInvalidBuild, rb.Build.Name)
	}
	if spec.Strategy.Name == "" {
		return domain.BuildSpec{}, fmt.Errorf("%w: build %q has no strategy name", domain.ErrInvalidBuild, rb.Build.Name)
	}

	var strategyKind string
	switch spec.Strategy.StrategyKind {
	case bldResolution.StrategyKindCluster:
		strategyKind = "ClusterBuildStrategy"
	case bldResolution.StrategyKindNamespaced:
		strategyKind = "BuildStrategy"
	default:
		return domain.BuildSpec{}, fmt.Errorf("%w: build %q has unsupported strategy kind %q",
			domain.ErrInvalidBuild, rb.Build.Name, spec.Strategy.StrategyKind)
	}

	// ServiceAccount is optional — zero values are valid and mean Shipwright
	// uses the namespace default.
	var saName, saSecret string
	if spec.ServiceAccount != nil {
		saName = spec.ServiceAccount.Name
		saSecret = spec.ServiceAccount.Secret
	}

	return domain.BuildSpec{
		SourceURL:   spec.Source.URL,
		ContextDir:  spec.Source.ContextDir,
		Revision:    spec.Source.Revision,
		CloneSecret: spec.Source.CloneSecret,

		StrategyName: spec.Strategy.Name,
		StrategyKind: strategyKind,

		Image: spec.Image,

		ServiceAccountName:   saName,
		ServiceAccountSecret: saSecret,
	}, nil
}
