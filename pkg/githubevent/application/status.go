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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
)

// StatusWriter persists GitHubEvent provisioning outcomes to the CR status.
type StatusWriter struct {
	Client client.Client

	Log logr.Logger
}

// NewStatusWriter constructs a StatusWriter.
func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("github-status-writer"),
	}
}

// Write persists the provider result and any error to the GitHubEvent CR.
// A non-nil runErr forces the Phase to "Failed".
func (w *StatusWriter) Write(ctx context.Context, githubevent *eventsv1alpha1.GitHubEvent, result domain.GitHubEventResult, runErr error) error {

	log := w.Log.WithValues("githubevent", githubevent.Name, "namespace", githubevent.Namespace)

	// ---------------------------------------------------------------------
	// 1. Write CONTRACT status (authoritative, user-facing)
	// ---------------------------------------------------------------------
	contractStatus := domain.GitHubEventStatus{
		Triggered: result.Triggered,
		Success:   result.Success,
		Message:   result.Message,
	}

	if runErr != nil {
		contractStatus.Success = false
		contractStatus.Message = runErr.Error()
	}

	raw, err := json.Marshal(contractStatus)
	if err != nil {
		log.Error(err, "failed to marshal githubevent contract status")
		return err
	}

	githubevent.Status.Contract = runtime.RawExtension{Raw: raw}
	log.Info("contract status written", "triggered", contractStatus.Triggered, "success", contractStatus.Success, "message", contractStatus.Message)

	// ---------------------------------------------------------------------
	// 2. Conditions (kubectl describe alignment)
	// ---------------------------------------------------------------------
	now := metav1.NewTime(time.Now())
	var condition metav1.Condition

	switch {
	case result.Triggered:
		condition = metav1.Condition{Type: "GitHubEventReady", Status: metav1.ConditionTrue, Reason: "GitHubEventReady", Message: "GitHubEvent Ready for next steps", LastTransitionTime: now}
	case runErr != nil:
		condition = metav1.Condition{Type: "GitHubEventFailed", Status: metav1.ConditionFalse, Reason: "GitHubEventFailed", Message: runErr.Error(), LastTransitionTime: now}
	case result.Success:
		condition = metav1.Condition{Type: "GitHubEventProcess", Status: metav1.ConditionTrue, Reason: "GitHubEventProcessed", Message: contractStatus.Message, LastTransitionTime: now}
	case result.PayloadRecieved:
		condition = metav1.Condition{Type: "GitHubEventReceiving", Status: metav1.ConditionTrue, Reason: "GitHubEventPayloadObserved", Message: contractStatus.Message, LastTransitionTime: now}
	default:
		condition = metav1.Condition{Type: "GitHubEventProcess", Status: metav1.ConditionFalse, Reason: "BuildExecuteFail", Message: result.Message, LastTransitionTime: now}
	}

	githubevent.Status.Conditions = mergeCondition(
		githubevent.Status.Conditions,
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
	if err := w.Client.Status().Update(ctx, githubevent); err != nil {
		log.Error(err, "failed to persist githubevent status")
		return err
	}

	log.Info("githubevent status persisted successfully")
	return nil
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
