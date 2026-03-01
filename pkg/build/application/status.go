package application

import (
	"context"
	"encoding/json"
	"time"

	buildv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("build-status-writer"),
	}
}

func (w *StatusWriter) Write(
	ctx context.Context,
	build *buildv1.Build,
	result domain.BuildResult,
	runErr error,
) error {

	log := w.Log.WithValues(
		"build", build.Name,
		"namespace", build.Namespace,
		"executionRef", result.ExecutionRef,
		"buildHash", result.BuildHash,
	)

	// ---------------------------------------------------------------------
	// 1. Write CONTRACT status (authoritative, user-facing)
	// ---------------------------------------------------------------------
	contractStatus := domain.BuildStatus{
		Triggered:    result.Triggered,
		Success:      result.Success,
		Message:      result.Message,
		ExecutionRef: result.ExecutionRef,
		BuildHash:    result.BuildHash,
	}

	if runErr != nil {
		contractStatus.Success = false
		contractStatus.Message = runErr.Error()
	}

	raw, err := json.Marshal(contractStatus)
	if err != nil {
		log.Error(err, "failed to marshal build contract status")
		return err
	}

	build.Status.Contract = runtime.RawExtension{Raw: raw}

	log.Info("contract status written",
		"triggered", contractStatus.Triggered,
		"success", contractStatus.Success,
		"message", contractStatus.Message,
	)

	// ---------------------------------------------------------------------
	// 2. Conditions (kubectl describe alignment)
	// ---------------------------------------------------------------------
	now := metav1.NewTime(time.Now())
	var condition metav1.Condition

	switch {
	case result.Triggered:
		condition = metav1.Condition{
			Type:               "Running",
			Status:             metav1.ConditionTrue,
			Reason:             "BuildTriggered",
			Message:            "Build execution started",
			LastTransitionTime: now,
		}
		log.Info("build execution started")

	case runErr != nil:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionFalse,
			Reason:             "BuildFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}
		log.Info("build failed", "error", runErr.Error())

	case result.Success:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionTrue,
			Reason:             "BuildSucceeded",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("build succeeded", "message", result.Message)

	default:
		condition = metav1.Condition{
			Type:               "Succeeded",
			Status:             metav1.ConditionFalse,
			Reason:             "BuildFailed",
			Message:            result.Message,
			LastTransitionTime: now,
		}
		log.Info("build failed", "message", result.Message)
	}

	build.Status.Conditions = mergeCondition(
		build.Status.Conditions,
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
	if err := w.Client.Status().Update(ctx, build); err != nil {
		log.Error(err, "failed to persist build status")
		return err
	}

	log.Info("build status persisted successfully")
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
