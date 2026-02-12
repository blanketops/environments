package application

import (
	"context"
	"encoding/json"
	"time"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/githubevent/domain"

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

func (w *StatusWriter) Write(
	ctx context.Context,
	ev *eventsv1alpha1.GitHubEvent,
	result domain.GitHubEventResult,
	runErr error,
) error {

	// ------------------------------------------------
	// 1. Build CONTRACT status (authoritative)
	// ------------------------------------------------
	contractStatus := domain.GitHubEventStatus{
		Accepted:     result.Accepted,
		Triggered:    result.Triggered,
		Message:      result.Message,
		TriggeredRef: result.TriggeredRef,
	}

	if runErr != nil {
		contractStatus.Accepted = false
		contractStatus.Triggered = false
		contractStatus.Message = runErr.Error()
	}

	raw, err := json.Marshal(contractStatus)
	if err != nil {
		return err
	}

	ev.Status.Contract = runtime.RawExtension{
		Raw: raw,
	}

	// ------------------------------------------------
	// 2. Kubernetes Conditions (observability only)
	// ------------------------------------------------
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

	ev.Status.Conditions = mergeCondition(
		ev.Status.Conditions,
		condition,
	)

	// ------------------------------------------------
	// 3. Persist status
	// ------------------------------------------------
	return w.Client.Status().Update(ctx, ev)
}

// mergeCondition inserts or replaces a condition of the same Type.
// This follows standard Kubernetes condition semantics.
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
