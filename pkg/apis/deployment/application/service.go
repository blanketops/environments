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

/*
Package application owns DeploymentService, the single entry point that
orchestrates a Deployment's reconciliation: build a DeploymentIntent from
resolved inputs (pkg/intent/deployment.IntentBuilder), execute it across the
Imperative/GitOps axis (pkg/apis/deployment/reconcile.ReconciliationExecutor),
then persist the outcome as CR status and conditions (StatusWriter).
Teardown mirrors the same Build-then-dispatch shape for deletion, minus the
status write.

This mirrors every other CR's application layer. mapper.go's Mapper /
MapResolvedToDomain is not part of that flow — DeploymentService goes
straight from ResolvedDeployment through IntentBuilder, never through this
Mapper's domain.DeploymentSpec — it predates the Intent layer and has no
callers.
*/
package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/blanketops/environments/pkg/apis/deployment/reconcile"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

// DeploymentService is the single entry point that orchestrates a
// Deployment's reconciliation: build a DeploymentIntent, execute it, then
// persist the outcome as CR status and conditions.
type DeploymentService struct {
	intentBuilder          *intent.IntentBuilder
	status                 *StatusWriter
	reconciliationExecutor *reconcile.ReconciliationExecutor
	log                    logr.Logger
}

// NewDeploymentService constructs a DeploymentService from its
// collaborators.
func NewDeploymentService(
	intentBuilder *intent.IntentBuilder,
	status *StatusWriter,
	reconciliationExecutor *reconcile.ReconciliationExecutor,
	log logr.Logger) *DeploymentService {
	return &DeploymentService{
		intentBuilder:          intentBuilder,
		status:                 status,
		reconciliationExecutor: reconciliationExecutor,
		log:                    log,
	}
}

// Reconcile builds a DeploymentIntent from resolved, executes it via the
// reconciliation executor, and writes the resulting status onto the CR.
//
// A failed execution is both recorded on the CR and returned. Returning only
// the result of the status write would tell the caller a deployment that
// could not be applied had succeeded.
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
	// A ServiceUnit that cannot be applied does not abort the execution: it
	// comes back in the result, not as an error. Either way it is a failure.
	if execErr == nil {
		execErr = result.Failure()
	}

	if err := s.status.WriteDeploymentResult(
		ctx,
		resolved.Deployment,
		result,
		execErr,
	); err != nil {
		if execErr != nil {
			return errors.Join(execErr, fmt.Errorf("write deployment status: %w", err))
		}
		return err
	}
	return execErr
}

// Teardown removes what Reconcile applied for the Deployment.
//
// It needs the resolved Deployment only. What was applied for each
// ServiceUnit is named after it, and the names are in the Deployment's
// contract, so teardown does not depend on the ServiceUnits still existing
// or resolving. The serviceUnits argument is kept for callers written against
// the earlier signature and is not used.
func (s *DeploymentService) Teardown(
	ctx context.Context,
	resolved *deploymentResolution.ResolvedDeployment,
	_ []serviceunitResolution.ResolvedServiceUnit,
	log logr.Logger,
) error {

	intent, err := s.intentBuilder.BuildTeardown(resolved)
	if err != nil {
		return err
	}
	log.Info("intent built for teardown",
		"manifestsRepoNil", intent.ManifestsRepo == nil,
		"reconciliationStrategy", intent.ReconciliationStrategy,
	)

	return s.reconciliationExecutor.Teardown(ctx, resolved.Deployment, intent)
}
