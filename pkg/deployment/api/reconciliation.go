package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
)

type ReconciliationExecutor struct {
	RuntimeProvider *RuntimeProvider
	Kustomizer      *KustomizeStrategyProvider
	Log             logr.Logger
}

func NewReconciliationExecutor(
	runtime *RuntimeProvider,
	kust *KustomizeStrategyProvider,
	log logr.Logger,
) *ReconciliationExecutor {
	return &ReconciliationExecutor{
		RuntimeProvider: runtime,
		Kustomizer:      kust,
		Log:             log,
	}
}

// Execute(ctx context.Context, sourceCR *environmentv1.Deployment, rIntent *intent.DeploymentIntent)

func (r *ReconciliationExecutor) Execute(
	ctx context.Context,
	sourceCR *environmentv1.Deployment,
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
