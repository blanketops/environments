package application

import (
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments/resolution/serviceunit"
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
