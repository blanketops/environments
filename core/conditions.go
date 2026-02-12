package core

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constants for consistent condition statuses across the operator.
const (
	ConditionTrue    = metav1.ConditionTrue
	ConditionFalse   = metav1.ConditionFalse
	ConditionUnknown = metav1.ConditionUnknown
)

// SetCondition adds or updates a condition in the CR status slice.
// If a condition of the same type exists, it updates the status, reason, and message.
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())

	// If no slice exists, create one
	if *conditions == nil {
		*conditions = []metav1.Condition{}
	}

	for i, cond := range *conditions {
		if cond.Type == conditionType {
			(*conditions)[i].Status = status
			(*conditions)[i].Reason = reason
			(*conditions)[i].Message = message
			(*conditions)[i].LastTransitionTime = now
			return
		}
	}

	// Otherwise, append new condition
	newCond := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	}
	*conditions = append(*conditions, newCond)
}

// GetCondition returns a pointer to a condition by type, if it exists.
func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// HasCondition checks if a condition of a given type and status exists.
func HasCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) bool {
	for _, cond := range conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}
