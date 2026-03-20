package intent

import (
	"time"

	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments/resolution/serviceunit"
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
