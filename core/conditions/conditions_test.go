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

package conditions

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetCondition_AppendsOnNilSlice(t *testing.T) {
	var conds []metav1.Condition

	SetCondition(&conds, "BuildResolved", ConditionTrue, "Resolved", "all good")

	if len(conds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conds))
	}
	if conds[0].Type != "BuildResolved" || conds[0].Status != ConditionTrue {
		t.Fatalf("unexpected condition: %+v", conds[0])
	}
	if conds[0].LastTransitionTime.IsZero() {
		t.Fatal("expected LastTransitionTime to be set")
	}
}

func TestSetCondition_AppendsNewTypeAlongsideExisting(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "BuildResolved", Status: ConditionTrue, Reason: "Resolved", Message: "ok"},
	}

	SetCondition(&conds, "BuildTriggered", ConditionFalse, "TriggerFailed", "boom")

	if len(conds) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(conds))
	}
	if conds[0].Type != "BuildResolved" {
		t.Fatal("expected existing condition to remain untouched")
	}
	if conds[1].Type != "BuildTriggered" || conds[1].Status != ConditionFalse {
		t.Fatalf("unexpected new condition: %+v", conds[1])
	}
}

func TestSetCondition_UpdatesExistingTypeInPlace(t *testing.T) {
	original := metav1.NewTime(metav1.Now().Add(-time.Hour))
	conds := []metav1.Condition{
		{Type: "BuildResolved", Status: ConditionFalse, Reason: "Pending", Message: "waiting", LastTransitionTime: original},
	}

	SetCondition(&conds, "BuildResolved", ConditionTrue, "Resolved", "all good")

	if len(conds) != 1 {
		t.Fatalf("expected update in place, not append — got %d conditions", len(conds))
	}
	got := conds[0]
	if got.Status != ConditionTrue || got.Reason != "Resolved" || got.Message != "all good" {
		t.Fatalf("unexpected condition after update: %+v", got)
	}
	if !got.LastTransitionTime.After(original.Time) {
		t.Fatal("expected LastTransitionTime to advance on update")
	}
}

func TestGetCondition_Found(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "BuildResolved", Status: ConditionTrue},
		{Type: "BuildTriggered", Status: ConditionFalse},
	}

	got := GetCondition(conds, "BuildTriggered")
	if got == nil {
		t.Fatal("expected to find BuildTriggered condition")
	}
	if got.Status != ConditionFalse {
		t.Fatalf("unexpected status: %v", got.Status)
	}

	// The returned pointer aliases the slice.
	got.Status = ConditionTrue
	if conds[1].Status != ConditionTrue {
		t.Fatal("expected GetCondition's pointer to alias the underlying slice")
	}
}

func TestGetCondition_NotFound(t *testing.T) {
	conds := []metav1.Condition{{Type: "BuildResolved", Status: ConditionTrue}}

	if got := GetCondition(conds, "Nonexistent"); got != nil {
		t.Fatalf("expected nil for missing condition type, got %+v", got)
	}
}

func TestHasCondition(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "BuildResolved", Status: ConditionTrue},
		{Type: "BuildTriggered", Status: ConditionFalse},
	}

	if !HasCondition(conds, "BuildResolved", ConditionTrue) {
		t.Fatal("expected HasCondition to find matching type+status")
	}
	if HasCondition(conds, "BuildResolved", ConditionFalse) {
		t.Fatal("expected HasCondition to return false when status doesn't match")
	}
	if HasCondition(conds, "Nonexistent", ConditionTrue) {
		t.Fatal("expected HasCondition to return false when type doesn't exist")
	}
}
