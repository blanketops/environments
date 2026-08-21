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
Package reconcile dispatches a DeploymentIntent across the Imperative-vs-
GitOps axis: apply the resolved workload live via
pkg/apis/deployment/strategy.RuntimeProvider, or commit manifests for Flux
to reconcile via pkg/apis/deployment/api.KustomizeStrategyProvider.

ReconciliationExecutor is the single entry point application.DeploymentService
calls once intent has been built — it does not know about ServiceUnits,
runtimes, or rollout strategies, only which reconciliation mode the intent
requested (intent.ManifestsRepo == nil selects imperative; otherwise the
mode is intent.ReconciliationStrategy).
*/
package reconcile

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/deployment/api"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	"github.com/blanketops/environments/pkg/apis/deployment/strategy"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

// ReconciliationExecutor dispatches on the DeploymentIntent's
// ReconciliationStrategy — imperative (apply live via a strategy.RuntimeProvider)
// vs GitOps (commit manifests via api.KustomizeStrategyProvider).
type ReconciliationExecutor struct {
	RuntimeProvider *strategy.RuntimeProvider
	Kustomizer      *api.KustomizeStrategyProvider
	Log             logr.Logger
}

// NewReconciliationExecutor constructs a ReconciliationExecutor.
func NewReconciliationExecutor(
	runtime *strategy.RuntimeProvider,
	kust *api.KustomizeStrategyProvider,
	log logr.Logger,
) *ReconciliationExecutor {
	return &ReconciliationExecutor{
		RuntimeProvider: runtime,
		Kustomizer:      kust,
		Log:             log,
	}
}

// Execute dispatches rIntent to the imperative or GitOps path based on
// rIntent.ManifestsRepo/ReconciliationStrategy.
func (r *ReconciliationExecutor) Execute(
	ctx context.Context,
	sourceCR *environmentv1alpha1.Deployment,
	rIntent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	if rIntent == nil {
		return nil, fmt.Errorf("nil deployment intent")
	}

	// ------------------------------------------------
	// Imperative Mode (No GitOps)
	// ------------------------------------------------
	if rIntent.ManifestsRepo == nil {
		r.Log.Info("Executing imperative reconciliation")
		return r.RuntimeProvider.Execute(ctx, rIntent)
	}

	// ------------------------------------------------
	// GitOps Mode
	// ------------------------------------------------
	switch rIntent.ReconciliationStrategy {

	case intent.ReconciliationKustomize:

		r.Log.Info("Executing GitOps reconciliation (kustomize)",
			"deployment", rIntent.Name,
		)

		repo := rIntent.ManifestsRepo

		err := r.Kustomizer.ReconcileKustomization(
			ctx,
			sourceCR, // ← real CR
			rIntent,
			repo.URL,
			repo.Ref.Commit,
			repo.Path,
		)

		if err != nil {
			return nil, err
		}

		// GitOps does not directly apply workloads.
		// Flux will reconcile.
		// So we return a synthetic DeploymentResult.

		return &domain.DeploymentResult{
			Phase:        domain.DeploymentPhase("Reconciling"),
			Runtime:      domain.Runtime(rIntent.Runtime),
			Strategy:     domain.Strategy(rIntent.Strategy),
			ServiceUnits: []domain.ServiceUnitResult{},
		}, nil

	case intent.ReconciliationHelm:
		return nil, fmt.Errorf("helm reconciliation not implemented")

	default:
		return nil, fmt.Errorf(
			"unsupported reconciliation strategy: %s",
			rIntent.ReconciliationStrategy,
		)
	}
}

// Teardown reverses Execute: deletes whatever the imperative or GitOps path
// applied for rIntent, dispatching on the same axis Execute uses.
func (r *ReconciliationExecutor) Teardown(
	ctx context.Context,
	sourceCR *environmentv1alpha1.Deployment,
	rIntent *intent.DeploymentIntent,
) error {

	if rIntent == nil {
		return fmt.Errorf("nil deployment intent")
	}

	// ------------------------------------------------
	// Imperative Mode (No GitOps)
	// ------------------------------------------------
	if rIntent.ManifestsRepo == nil {
		r.Log.Info("Tearing down imperative reconciliation")
		return r.RuntimeProvider.Teardown(ctx, rIntent)
	}

	// ------------------------------------------------
	// GitOps Mode
	// ------------------------------------------------
	switch rIntent.ReconciliationStrategy {

	case intent.ReconciliationKustomize:

		r.Log.Info("Tearing down GitOps reconciliation (kustomize)",
			"deployment", rIntent.Name,
		)

		env, ok := sourceCR.Labels["environments.blanketops.dev/type"]
		if !ok || env == "" {
			return fmt.Errorf("deployment CR must define label environments.blanketops.dev/type")
		}

		repo := rIntent.ManifestsRepo
		return r.Kustomizer.Teardown(ctx, sourceCR, rIntent, repo.URL, repo.Ref.Commit, env)

	case intent.ReconciliationHelm:
		return fmt.Errorf("helm reconciliation not implemented")

	default:
		return fmt.Errorf(
			"unsupported reconciliation strategy: %s",
			rIntent.ReconciliationStrategy,
		)
	}
}
