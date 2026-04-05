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

	buildtriggerv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusWriter struct {
	Client client.Client
}

func NewStatusWriter(c client.Client) *StatusWriter {
	return &StatusWriter{Client: c}
}

// Write persists a BuildTrigger decision.
//
// CONTRACT:
// - Domain status is authoritative
// - CR status is persistence + observability only
func (w *StatusWriter) Write(
	ctx context.Context,
	cr *buildtriggerv1.BuildTrigger,
	status domain.BuildTriggerStatus,
	runErr error,
) error {

	// ------------------------------------------------
	// 1. CONTRACT status (authoritative)
	// ------------------------------------------------
	if runErr != nil {
		status.Accepted = false
		status.Triggered = false
		status.Message = runErr.Error()
	}

	raw, err := json.Marshal(status)
	if err != nil {
		return err
	}

	cr.Status.Contract = runtime.RawExtension{
		Raw: raw,
	}

	// ------------------------------------------------
	// 2. Kubernetes Conditions (observability)
	// ------------------------------------------------
	now := metav1.NewTime(time.Now())

	var condition metav1.Condition

	switch {
	case runErr != nil:
		condition = metav1.Condition{
			Type:               "Accepted",
			Status:             metav1.ConditionFalse,
			Reason:             "SystemError",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}

	case !status.Accepted:
		condition = metav1.Condition{
			Type:               "Accepted",
			Status:             metav1.ConditionFalse,
			Reason:             "Rejected",
			Message:            status.Message,
			LastTransitionTime: now,
		}

	case status.Accepted && !status.Triggered:
		condition = metav1.Condition{
			Type:               "Triggered",
			Status:             metav1.ConditionFalse,
			Reason:             "NoExecution",
			Message:            status.Message,
			LastTransitionTime: now,
		}

	case status.Accepted && status.Triggered:
		condition = metav1.Condition{
			Type:               "Triggered",
			Status:             metav1.ConditionTrue,
			Reason:             "BuildTriggered",
			Message:            status.Message,
			LastTransitionTime: now,
		}
	}

	cr.Status.Conditions = mergeCondition(
		cr.Status.Conditions,
		condition,
	)

	// ------------------------------------------------
	// 3. Persist status
	// ------------------------------------------------
	return w.Client.Status().Update(ctx, cr)
}

// mergeCondition inserts or replaces a condition of the same Type.
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
