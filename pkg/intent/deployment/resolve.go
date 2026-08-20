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
	"time"

	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"

	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

// ResolveDeploymentIntent is an older, partial DeploymentIntent constructor
// that never learned Runtime/Strategy/ReconciliationStrategy/ManifestsRepo.
// It has no callers in DeploymentService's path — superseded by
// IntentBuilder.Build, not a second supported entry point.
func ResolveDeploymentIntent(
	deploy *deploymentResolution.ResolvedDeployment,
	serviceUnits map[string]*serviceunitResolution.ResolvedServiceUnit,
) (*DeploymentIntent, error) {

	if deploy == nil || deploy.Spec == nil {
		return nil, ErrInvalidDeployment("nil resolved deployment")
	}

	intent := &DeploymentIntent{
		Name:      deploy.Deployment.Name,
		Namespace: deploy.Deployment.Namespace,

		GeneratedAt: time.Now(),
	}

	for _, suName := range deploy.Spec.ServiceUnits {

		su, ok := serviceUnits[suName]
		if !ok {
			return nil, ErrServiceUnitNotFound(suName)
		}

		suIntent, err := serviceunitIntent.ResolveServiceUnitIntent(su)
		if err != nil {
			return nil, err
		}

		intent.ServiceUnits = append(intent.ServiceUnits, *suIntent)
	}

	return intent, nil
}
