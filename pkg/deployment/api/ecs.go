package api

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
	"k8s.io/apimachinery/pkg/runtime"
)

type ECSReconciler struct {
	Log    logr.Logger
	Scheme *runtime.Scheme
	// TODO: add AWS SDK clients here
}

func NewECSReconciler(scheme *runtime.Scheme, log logr.Logger) *ECSReconciler {
	return &ECSReconciler{
		Log:    log,
		Scheme: scheme,
	}
}

// Reconcile executes a ServiceUnitIntent on ECS
func (r *ECSReconciler) Reconcile(
	ctx context.Context,
	su intent.ServiceUnitIntent,
) (intent.ServiceUnitResult, error) {

	r.Log.Info("Reconciling ECS ServiceUnit",
		"name", su.Name,
		"image", su.Image,
	)

	// --- Task definition ---
	if err := r.registerTaskDefinition(ctx, su); err != nil {
		return intent.ServiceUnitResult{
			Name:  su.Name,
			Phase: intent.ServiceUnitFailed,
			Error: err.Error(),
		}, err
	}

	// --- ECS Service ---
	if err := r.ensureService(ctx, su); err != nil {
		return intent.ServiceUnitResult{
			Name:  su.Name,
			Phase: intent.ServiceUnitFailed,
			Error: err.Error(),
		}, err
	}

	return intent.ServiceUnitResult{
		Name:    su.Name,
		Phase:   intent.ServiceUnitReady,
		Image:   su.Image,
		Runtime: intent.RuntimeECS,
	}, nil
}

func (r *ECSReconciler) registerTaskDefinition(
	ctx context.Context,
	su intent.ServiceUnitIntent,
) error {
	r.Log.Info("Registering ECS TaskDefinition (stub)", "serviceUnit", su.Name)
	return nil
}

func (r *ECSReconciler) ensureService(
	ctx context.Context,
	su intent.ServiceUnitIntent,
) error {
	r.Log.Info("Ensuring ECS Service (stub)", "serviceUnit", su.Name)
	return nil
}
