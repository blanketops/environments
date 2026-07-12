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
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	env1alpha1 "github.com/BlanketOps/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
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

// WriteDeploymentResult merges contract status and conditions onto the
// Deployment CR and persists. Uses RetryOnConflict + re-fetch on each attempt
// — concurrent writers may bump resourceVersion between the reconciler's
// initial fetch and this write. Re-fetching guarantees we always write to
// the latest version. Contract and conditions are re-applied on every retry.
func (w *StatusWriter) WriteDeploymentResult(
	ctx context.Context,
	depl *env1alpha1.Deployment,
	result *domain.DeploymentResult,
	runErr error,
) error {
	log := w.Log.WithValues("deployment", depl.Name, "namespace", depl.Namespace)

	// Derive condition outside the retry loop — idempotent, no API calls.
	condition, ok := w.deriveCondition(result, runErr, log)
	if !ok {
		return nil
	}

	// Marshal contract once — same value on every retry attempt.
	var contractRaw []byte
	if result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			log.Error(err, "failed to marshal deployment contract status")
			return err
		}
		contractRaw = raw
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &env1alpha1.Deployment{}
		if err := w.Client.Get(ctx, client.ObjectKeyFromObject(depl), latest); err != nil {
			log.Error(err, "failed to re-fetch deployment before status write")
			return err
		}

		// Re-apply contract status on every attempt.
		if contractRaw != nil {
			latest.Status.Contract = runtime.RawExtension{Raw: contractRaw}
			log.Info("contract status written", "phase", result.Phase, "message", result.Message)
		}

		// Re-apply condition on every attempt.
		latest.Status.Conditions = mergeCondition(latest.Status.Conditions, condition)
		log.Info("condition updated",
			"type", condition.Type,
			"status", condition.Status,
			"reason", condition.Reason,
		)

		if err := w.Client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "failed to persist deployment status")
			return err
		}

		log.Info("deployment status persisted successfully")
		return nil
	})
}

// deriveCondition computes the metav1.Condition from the deployment result
// and any run error. Returns (condition, false) when there is nothing to write.
func (w *StatusWriter) deriveCondition(
	result *domain.DeploymentResult,
	runErr error,
	log logr.Logger,
) (metav1.Condition, bool) {
	now := metav1.NewTime(time.Now())

	switch {
	case runErr != nil:
		log.Info("deployment failed", "error", runErr.Error())
		return metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "DeploymentFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}, true

	case result != nil && result.Phase == domain.DeploymentPhase("Ready"):
		log.Info("deployment ready", "message", result.Message)
		return metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReady",
			Message:            result.Message,
			LastTransitionTime: now,
		}, true

	case result != nil:
		log.Info("deployment reconciling", "phase", result.Phase)
		return metav1.Condition{
			Type:               "Reconciling",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReconciling",
			Message:            result.Message,
			LastTransitionTime: now,
		}, true

	default:
		return metav1.Condition{}, false
	}
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
