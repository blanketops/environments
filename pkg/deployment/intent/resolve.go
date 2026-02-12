package intent

import (
	"fmt"
	"time"

	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
	deploymentResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/serviceunit"
)

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

		Runtime: mapRuntime(deploy.Spec.Runtime),

		GeneratedAt: time.Now(),
	}

	for _, suName := range deploy.Spec.ServiceUnits {

		su, ok := serviceUnits[suName]
		if !ok {
			return nil, ErrServiceUnitNotFound(suName)
		}

		suIntent, err := ResolveServiceUnitIntent(su)
		if err != nil {
			return nil, err
		}

		intent.ServiceUnits = append(intent.ServiceUnits, *suIntent)
	}

	return intent, nil
}

func mapRuntime(rt contractv1.DeploymentRuntime) Runtime {
	switch rt {

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER:
		return RuntimeKubernetes

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE:
		return RuntimeKnative

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_AWS_ECS:
		return RuntimeECS

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_WASM:
		return RuntimeWASM

	case contractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_AZURE_CONTAINER:
		return RuntimeAzure

	default:
		panic(fmt.Sprintf(
			"unsupported deployment runtime %q (resolver bug)",
			rt,
		))
	}
}
