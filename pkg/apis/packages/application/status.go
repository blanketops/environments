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

	env1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(
	c client.Client,
	log logr.Logger,
) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("package-status-writer"),
	}
}

func (w *StatusWriter) Write(
	ctx context.Context,
	pkg *env1alpha1.Package,
	result *domain.PackageResult,
	runErr error,
) error {

	log := w.Log.WithValues(
		"package", pkg.Name,
		"namespace", pkg.Namespace,
		"phase", result.Phase,
		"desiredCommit", result.DesiredCommit,
	)

	// ---------------------------------------------------------------------
	// 1. CONTRACT STATUS (authoritative, user-facing)
	// ---------------------------------------------------------------------

	contractStatus := domain.PackageStatus{
		Phase:         result.Phase,
		Success:       result.Success,
		Message:       result.Message,
		DesiredRef:    result.DesiredRef,
		DesiredCommit: result.DesiredCommit,
		KappAction:    result.Kapp.Action,
		Diff:          result.Kapp.Diff,
		Executed:      result.Kapp.Executed,
		StartedAt:     result.StartedAt,
		FinishedAt:    result.FinishedAt,
	}

	if runErr != nil {
		contractStatus.Success = false
		contractStatus.Message = runErr.Error()
	}

	raw, err := json.Marshal(contractStatus)
	if err != nil {
		log.Error(err, "failed to marshal package contract status")
		return err
	}

	pkg.Status.Contract = runtime.RawExtension{Raw: raw}

	log.Info(
		"contract status written",
		"success", contractStatus.Success,
		"phase", contractStatus.Phase,
		"action", contractStatus.KappAction,
	)

	// ---------------------------------------------------------------------
	// 2. CONDITIONS (kubectl describe alignment)
	// ---------------------------------------------------------------------

	now := metav1.NewTime(time.Now())
	var condition metav1.Condition

	switch {
	case result.Kapp.Executed && !result.Kapp.DeployFinished:
		condition = metav1.Condition{
			Type:               "Running",
			Status:             metav1.ConditionTrue,
			Reason:             "PackageDeploying",
			Message:            "Package deployment in progress",
			LastTransitionTime: now,
		}
		log.Info("package deployment started")

	case runErr != nil:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionFalse,
			Reason:             "PackageFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}
		log.Info("package failed", "error", runErr.Error())

	case result.Success:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionTrue,
			Reason:             "PackageApplied",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("package succeeded", "message", result.Message)

	default:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionFalse,
			Reason:             "PackageFailed",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("package failed", "message", result.Message)
	}

	pkg.Status.Conditions = mergeCondition(
		pkg.Status.Conditions,
		condition,
	)

	log.Info(
		"condition updated",
		"type", condition.Type,
		"status", condition.Status,
		"reason", condition.Reason,
	)

	// ---------------------------------------------------------------------
	// 3. Persist
	// ---------------------------------------------------------------------

	if err := w.Client.Status().Update(ctx, pkg); err != nil {
		log.Error(err, "failed to persist package status")
		return err
	}

	log.Info("package status persisted successfully")
	return nil
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

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
