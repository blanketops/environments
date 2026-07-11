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
	"context"
	"fmt"
	"time"

	deploymentResolution "github.com/BlanketOps/environments/resolution/deployment"
	serviceunitResolution "github.com/BlanketOps/environments/resolution/serviceunit"
	intent "github.com/BlanketOps/environments/pkg/intent/deployment"
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
) (*intent.DeploymentIntent, error) {

	if depl == nil || depl.Spec == nil {
		return nil, fmt.Errorf("nil ResolvedDeployment (resolver bug)")
	}

	// ------------------------------------------------------------
	// Resolve ServiceUnit intents
	// ------------------------------------------------------------

	units := make([]intent.ServiceUnitIntent, 0, len(serviceUnits))

	for _, su := range serviceUnits {
		suIntent, err := intent.ResolveServiceUnitIntent(&su)
		if err != nil {
			return nil, fmt.Errorf(
				"serviceunit %s: %w",
				su.ServiceUnit.Name,
				err,
			)
		}
		units = append(units, *suIntent)
	}

	// ------------------------------------------------------------
	// Build Deployment intent (pure semantic)
	// ------------------------------------------------------------

	return &intent.DeploymentIntent{
		Name:         depl.Deployment.Name,
		Namespace:    depl.Deployment.Namespace,
		Runtime:      intent.Runtime(depl.Spec.Runtime),
		Strategy:     intent.Strategy(depl.Spec.Strategy),
		ServiceUnits: units,
		GeneratedAt:  time.Now(),

		Source: depl.Deployment, // ← REQUIRED
	}, nil
}
