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

package api

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"

	intent "github.com/BlanketOps/environments/pkg/intent/deployment"
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
