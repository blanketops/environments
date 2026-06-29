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

	"github.com/go-logr/logr"

	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/deployment/api"
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
