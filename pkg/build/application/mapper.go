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

The Mapper enforces the resolution contract: it panics on fields the resolver
guarantees to be present (Source.URL, Strategy.Name) so that resolver bugs
surface loudly rather than silently producing empty Shipwright specs.

Optional fields (ServiceAccount, CloneSecret) are passed through verbatim —
the Mapper never invents defaults or modifies intent.
*/
package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	bldResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
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
// Panics on resolver invariant violations (empty SourceURL or StrategyName) —
// these indicate a resolver bug, not a user error, and must not be silently
// swallowed. All other fields are mapped verbatim.
//
// StrategyKind is hardcoded to "ClusterBuildStrategy" — BlanketOps platform
// strategies are always cluster-scoped. Namespace-scoped strategies are not
// currently supported.
func (Mapper) MapResolvedToDomain(rb *bldResolution.ResolvedBuild) domain.BuildSpec {
	spec := rb.Spec

	// Resolver invariants — panic loudly on violation.
	if spec.Source.URL == "" {
		panic(fmt.Sprintf("resolved build %q has empty Source.URL (resolver bug)", rb.Build.Name))
	}
	if spec.Strategy.Name == "" {
		panic(fmt.Sprintf("resolved build %q has empty Strategy.Name (resolver bug)", rb.Build.Name))
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
		StrategyKind: "ClusterBuildStrategy",

		Image: spec.Image,

		ServiceAccountName:   saName,
		ServiceAccountSecret: saSecret,
	}
}
