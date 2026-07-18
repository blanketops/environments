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

package deployment

import (
	"context"
	"fmt"
	"time"

	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

type IntentBuilder struct{}

func NewIntentBuilder() *IntentBuilder {
	return &IntentBuilder{}
}

// Build constructs a DeploymentIntent from fully RESOLVED inputs.
//
// CONTRACT:
// - Inputs are already validated and normalized
// - No Kubernetes types allowed
// - No string-to-enum logic allowed
// - Any invalid state is a resolver bug
func (b *IntentBuilder) Build(
	ctx context.Context,
	depl *deploymentResolution.ResolvedDeployment,
	serviceUnits []serviceunitResolution.ResolvedServiceUnit,
) (*DeploymentIntent, error) {

	if depl == nil || depl.Spec == nil {
		return nil, fmt.Errorf("nil ResolvedDeployment (resolver bug)")
	}

	// ---------------------------------------------------------------------
	// Resolve ServiceUnit intents (already resolved inputs)
	// ---------------------------------------------------------------------

	units := make([]serviceunitIntent.ServiceUnitIntent, 0, len(serviceUnits))

	for _, su := range serviceUnits {
		suIntent, err := serviceunitIntent.ResolveServiceUnitIntent(&su)
		if err != nil {
			return nil, fmt.Errorf(
				"serviceunit %s: %w",
				su.ServiceUnit.Name,
				err,
			)
		}
		units = append(units, *suIntent)
	}
	var manifestsRepo *ManifestsRepo

	if depl.Spec.ManifestsRepo != nil {
		manifestsRepo = &ManifestsRepo{
			URL:         depl.Spec.ManifestsRepo.URL,
			Path:        depl.Spec.ManifestsRepo.Path,
			CloneSecret: depl.Spec.ManifestsRepo.CloneSecret,
		}
	}

	// ---------------------------------------------------------------------
	// Build Deployment intent (semantic, domain-level)
	// ---------------------------------------------------------------------

	return &DeploymentIntent{
		Name:      depl.Deployment.Name,
		Namespace: depl.Deployment.Namespace,

		Runtime:  Runtime(depl.Spec.Runtime),
		Strategy: Strategy(depl.Spec.Strategy),

		ReconciliationStrategy: ReconciliationStrategy(
			depl.Spec.ReconciliationStrategy,
		),

		ManifestsRepo: manifestsRepo,

		ServiceUnits: units,

		GeneratedAt: time.Now(),
	}, nil

}
