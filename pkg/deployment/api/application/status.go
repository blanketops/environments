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
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	env1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/domain"
)

type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("deployment-status-writer"),
	}
}

func (w *StatusWriter) WriteDeploymentResult(
	ctx context.Context,
	depl *env1alpha1.Deployment,
	result *domain.DeploymentResult,
	runErr error,
) error {

	log := w.Log.WithValues(
		"deployment", depl.Name,
		"namespace", depl.Namespace,
	)

	// ---------------------------------------------------------------------
	// 1. Write CONTRACT status (authoritative, user-facing)
	// ---------------------------------------------------------------------
	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			log.Error(err, "failed to marshal deployment contract status")
			return err
		}

		depl.Status.Contract = runtime.RawExtension{Raw: raw}

		log.Info("contract status written",
			"phase", result.Phase,
			"message", result.Message,
		)
	}

	// ---------------------------------------------------------------------
	// 2. Conditions (kubectl / ecosystem alignment)
	// ---------------------------------------------------------------------
	now := metav1.NewTime(time.Now())
	var condition metav1.Condition

	switch {
	case runErr != nil:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "DeploymentFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}
		log.Info("deployment failed", "error", runErr.Error())

	case result != nil && result.Phase == domain.DeploymentPhase("Ready"):
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReady",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("deployment ready", "message", result.Message)

	case result != nil:
		condition = metav1.Condition{
			Type:               "Reconciling",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReconciling",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("deployment reconciling", "phase", result.Phase)

	default:
		// no-op: nothing to write yet
		return nil
	}

	depl.Status.Conditions = mergeCondition(
		depl.Status.Conditions,
		condition,
	)

	log.Info("condition updated",
		"type", condition.Type,
		"status", condition.Status,
		"reason", condition.Reason,
	)

	// ---------------------------------------------------------------------
	// 3. Persist
	// ---------------------------------------------------------------------
	if err := w.Client.Status().Update(ctx, depl); err != nil {
		log.Error(err, "failed to persist deployment status")
		return err
	}

	log.Info("deployment status persisted successfully")
	return nil
}

func mergeCondition(
	conds []metav1.Condition,
	newCond metav1.Condition,
) []metav1.Condition {

	for i, c := range conds {
		if c.Type == newCond.Type {
			conds[i] = newCond
			return conds
		}
	}
	return append(conds, newCond)
}
