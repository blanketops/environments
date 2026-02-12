package application

import (
	"context"
	"fmt"
	"time"

	deploymentResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/serviceunit"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/intent"
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

	// ---------------------------------------------------------------------
	// Resolve ServiceUnit intents (already resolved inputs)
	// ---------------------------------------------------------------------

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

	// ---------------------------------------------------------------------
	// Build Deployment intent (semantic, domain-level)
	// ---------------------------------------------------------------------

	return &intent.DeploymentIntent{
		Name:      depl.Deployment.Name,
		Namespace: depl.Deployment.Namespace,

		// 🔒 Semantic enum — already resolved
		Runtime: intent.Runtime(depl.Spec.Runtime),

		ServiceUnits: units,

		GeneratedAt: time.Now(),
	}, nil
}
