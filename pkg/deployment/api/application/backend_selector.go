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

	intent "github.com/ntlaletsi70/blanketops-environments/intent/deployment"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/api"
)

type BackendSelector struct {
	Kubernetes api.Provider
	Knative    api.Provider
	ECS        api.Provider
	Flux       api.Provider
}

func NewBackendSelector(
	k8s api.Provider,
	knative api.Provider,
	ecs api.Provider,
	flux api.Provider,
) *BackendSelector {
	return &BackendSelector{
		Kubernetes: k8s,
		Knative:    knative,
		ECS:        ecs,
		Flux:       flux,
	}
}
func (b *BackendSelector) ForIntent(
	deploymentIntent *intent.DeploymentIntent,
) api.Provider {

	switch deploymentIntent.Runtime {

	case intent.RuntimeKubernetes:
		switch deploymentIntent.Strategy {

		case intent.StrategyRolling:
			// GitOps-aware Kubernetes
			if deploymentIntent.ManifestsRepo != nil {
				return b.Flux
			}
			return b.Kubernetes

		case intent.StrategyBlueGreen:
			// Service-switch based orchestration on Kubernetes
			return b.Kubernetes

		case intent.StrategyCanary:
			// Guarded by IntentBuilder, but never silently route
			panic("canary strategy is not supported on kubernetes runtime")
		}

	case intent.RuntimeKnative:
		switch deploymentIntent.Strategy {

		case intent.StrategyRolling:
			return b.Knative

		case intent.StrategyCanary:
			// Native traffic splitting
			return b.Knative

		case intent.StrategyBlueGreen:
			panic("bluegreen strategy is not supported on knative runtime")
		}

	case intent.RuntimeECS:
		panic("deployment strategies are not yet supported on ecs runtime")

	default:
		panic(fmt.Sprintf(
			"unsupported runtime: %s",
			deploymentIntent.Runtime,
		))
	}

	panic(fmt.Sprintf(
		"no backend matched for runtime=%s strategy=%s",
		deploymentIntent.Runtime,
		deploymentIntent.Strategy,
	))
}
