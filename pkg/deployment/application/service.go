package application

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/api"
	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	serviceunitResolution "github.com/ntlaletsi70/blanketops-environments/resolution/serviceunit"
)

type DeploymentService struct {
	intentBuilder          *IntentBuilder
	status                 *StatusWriter
	reconciliationExecutor *api.ReconciliationExecutor
	log                    logr.Logger
}

func NewDeploymentService(
	intentBuilder *IntentBuilder,
	status *StatusWriter,
	reconciliationExecutor *api.ReconciliationExecutor,
	log logr.Logger) *DeploymentService {
	return &DeploymentService{
		intentBuilder:          intentBuilder,
		status:                 status,
		reconciliationExecutor: reconciliationExecutor,
		log:                    log,
	}
}

func (s *DeploymentService) Reconcile(
	ctx context.Context,
	resolved *deploymentResolution.ResolvedDeployment,
	serviceUnits []serviceunitResolution.ResolvedServiceUnit,
	log logr.Logger,
) error {

	// 1. Build intent
	intent, err := s.intentBuilder.Build(ctx, resolved, serviceUnits)
	if err != nil {
		return err
	}
	log.Info("intent built",
		"manifestsRepoNil", intent.ManifestsRepo == nil,
		"reconciliationStrategy", intent.ReconciliationStrategy,
	)

	// 2. Execute via reconciliation axis
	result, execErr := s.reconciliationExecutor.Execute(
		ctx,
		resolved.Deployment, // ← real CR
		intent,
	)

	// 3. Write status
	return s.status.WriteDeploymentResult(
		ctx,
		resolved.Deployment,
		result,
		execErr,
	)
}
