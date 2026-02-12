package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/api"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/intent"
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
