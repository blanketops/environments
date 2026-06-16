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
This file owns StatusWriter for the GitHubEvent domain — the component
responsible for persisting provider results back to the GitHubEvent CR.

Two writes happen in a single Status().Update call:

 1. Contract status — JSON-marshalled domain status written to Status.Contract.
    Authoritative and machine-readable.

 2. Conditions — a single metav1.Condition upserted for kubectl describe
    alignment and human observability.

StatusWriter is always called regardless of provider error so the CR always
reflects the latest outcome.
*/
package application

import (
	"context"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
)

// StatusWriter persists GitHubEvent provisioning outcomes to the CR status.
type StatusWriter struct {
	Client client.Client
}

// NewStatusWriter constructs a StatusWriter.
func NewStatusWriter(c client.Client) *StatusWriter {
	return &StatusWriter{Client: c}
}

// Write persists the provider result and any error to the GitHubEvent CR.
// A non-nil runErr marks the event as not accepted and not triggered.
func (w *StatusWriter) Write(
	ctx context.Context,
	ev *eventsv1alpha1.GitHubEvent,
	result domain.GitHubEventResult,
	runErr error,
) error {
	// Stage 1: contract status (authoritative, machine-readable).
	contractStatus := result.ToStatus()
	if runErr != nil {
		contractStatus.Accepted = false
		contractStatus.Triggered = false
		contractStatus.Message = runErr.Error()
	}

	raw, err := json.Marshal(contractStatus)
	if err != nil {
		return err
	}
	ev.Status.Contract = runtime.RawExtension{Raw: raw}

	// Stage 2: condition (kubectl describe / human observability).
	now := metav1.NewTime(time.Now())
	var condition metav1.Condition

	switch {
	case runErr != nil:
		condition = metav1.Condition{
			Type:               "Accepted",
			Status:             metav1.ConditionFalse,
			Reason:             "ProcessingFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}
	case !result.Accepted:
		condition = metav1.Condition{
			Type:               "Accepted",
			Status:             metav1.ConditionFalse,
			Reason:             "Rejected",
			Message:            result.Message,
			LastTransitionTime: now,
		}
	case result.Accepted && !result.Triggered:
		condition = metav1.Condition{
			Type:               "Triggered",
			Status:             metav1.ConditionFalse,
			Reason:             "NoAction",
			Message:            result.Message,
			LastTransitionTime: now,
		}
	case result.Accepted && result.Triggered:
		condition = metav1.Condition{
			Type:               "Triggered",
			Status:             metav1.ConditionTrue,
			Reason:             "EventProcessed",
			Message:            result.Message,
			LastTransitionTime: now,
		}
	}

	ev.Status.Conditions = mergeCondition(ev.Status.Conditions, condition)

	// Stage 3: persist — single status subresource update.
	return w.Client.Status().Update(ctx, ev)
}

// mergeCondition upserts newCond into conds by Type. Replaces in place when
// a condition of the same Type exists; appends otherwise.
func mergeCondition(conds []metav1.Condition, newCond metav1.Condition) []metav1.Condition {
	for i, c := range conds {
		if c.Type == newCond.Type {
			conds[i] = newCond
			return conds
		}
	}
	return append(conds, newCond)
}
