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
Package core defines the foundational contracts and primitives shared across
all BlanketOps Environments domains.

This file owns condition management. Conditions are the canonical mechanism
for communicating reconciliation state on a CR's status subresource. Every
domain writes conditions at each stage of its pipeline via SetCondition so
that operators and tooling can observe fine-grained progress without reading
logs.

Condition types are domain-defined (e.g. "BuildResolved", "BuildTriggered")
and written at each reconciliation stage. The three status aliases here
(ConditionTrue, ConditionFalse, ConditionUnknown) are re-exported from
metav1 so domain code imports only core and never reaches into k8s.io
directly for condition status values.
*/
package core

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Re-exported metav1 condition status constants. Domain code uses these
// aliases so condition writes read naturally at the call site:
//
//	core.SetCondition(&cr.Status.Conditions, "BuildResolved", core.ConditionTrue, ...)
const (
	ConditionTrue    = metav1.ConditionTrue
	ConditionFalse   = metav1.ConditionFalse
	ConditionUnknown = metav1.ConditionUnknown
)

// SetCondition upserts a condition into a CR's conditions slice.
//
// If a condition of the same Type already exists, its Status, Reason,
// Message, and LastTransitionTime are updated in place. If no matching
// condition exists, a new one is appended.
//
// Called by domains at each pipeline stage to record progress:
//
//	core.SetCondition(&build.Status.Conditions, "BuildResolved", core.ConditionTrue, "Resolved", "...")
//	core.SetCondition(&build.Status.Conditions, "BuildTriggered", core.ConditionFalse, "TriggerFailed", err.Error())
//
// The conditions pointer must not be nil; a nil slice is initialised
// on first write.
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())

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

	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}

// GetCondition returns a pointer to the condition matching conditionType,
// or nil if no such condition exists. The pointer is into the slice —
// callers must not retain it across mutations of the conditions slice.
func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// HasCondition reports whether a condition of the given type and status
// exists in the slice. Useful for predicate checks without needing the
// full condition struct.
//
//	if core.HasCondition(build.Status.Conditions, "BuildResolved", core.ConditionTrue) { ... }
func HasCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) bool {
	for _, cond := range conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}
