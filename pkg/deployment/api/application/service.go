package application

import (
	"context"

	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments/resolution/serviceunit"
)

type DeploymentService struct {
	intentBuilder *IntentBuilder
	status        *StatusWriter
	backend       *BackendSelector
}

func NewDeploymentService(
	intentBuilder *IntentBuilder,
	status *StatusWriter,
	backend *BackendSelector,
) *DeploymentService {
	return &DeploymentService{
		intentBuilder: intentBuilder,
		status:        status,
		backend:       backend,
	}
}

func (s *DeploymentService) Reconcile(
	ctx context.Context,
	resolved *deploymentResolution.ResolvedDeployment,
	serviceUnits []serviceunitResolution.ResolvedServiceUnit,
) error {

	// ------------------------------------------------
	// 1. Build intent from resolved inputs
	// ------------------------------------------------
	intent, err := s.intentBuilder.Build(ctx, resolved, serviceUnits)
	if err != nil {
		return err
	}

	// ------------------------------------------------
	// 2. Select backend
	// ------------------------------------------------

	provider := s.backend.ForIntent(intent)

	// ------------------------------------------------
	// 3. Execute deployment
	// ------------------------------------------------
	result, err := provider.Execute(ctx, intent)

	// ------------------------------------------------
	// 4. Write status (still against CR)
	// ------------------------------------------------
	return s.status.WriteDeploymentResult(
		ctx,
		resolved.Deployment,
		result,
		err,
	)
}
