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
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
)

func TestStatusWriter_Write_MergesAndUpdates(t *testing.T) {
	su := &environmentv1alpha1.ServiceUnit{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Status: environmentv1alpha1.ServiceUnitStatus{
			Conditions: []metav1.Condition{
				{Type: "ServiceUnitPending", Status: metav1.ConditionFalse, Reason: "ServiceUnitPending"},
			},
		},
	}
	c := newTestClient(t, su)
	w := NewStatusWriter(c, logr.Discard())

	// First write: upgrades the existing Pending condition to Ready in place,
	// and adds a brand new condition type alongside it.
	err := w.Write(context.Background(), su,
		metav1.Condition{Type: "ServiceUnitPending", Status: metav1.ConditionTrue, Reason: "Upgraded"},
		metav1.Condition{Type: "SomethingElse", Status: metav1.ConditionTrue, Reason: "New"},
	)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	var got environmentv1alpha1.ServiceUnit
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(su), &got); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if len(got.Status.Conditions) != 2 {
		t.Fatalf("len(conditions) = %d, want 2: %+v", len(got.Status.Conditions), got.Status.Conditions)
	}

	byType := map[string]metav1.Condition{}
	for _, cond := range got.Status.Conditions {
		byType[cond.Type] = cond
	}

	if byType["ServiceUnitPending"].Status != metav1.ConditionTrue || byType["ServiceUnitPending"].Reason != "Upgraded" {
		t.Errorf("ServiceUnitPending not upgraded in place: %+v", byType["ServiceUnitPending"])
	}
	if byType["SomethingElse"].Status != metav1.ConditionTrue {
		t.Errorf("SomethingElse not added: %+v", byType["SomethingElse"])
	}
}

func TestStatusWriter_Write_ObjectNotFound(t *testing.T) {
	c := newTestClient(t)
	w := NewStatusWriter(c, logr.Discard())

	su := &environmentv1alpha1.ServiceUnit{
		ObjectMeta: metav1.ObjectMeta{Name: "missing", Namespace: "default"},
	}

	err := w.Write(context.Background(), su, metav1.Condition{Type: "X", Status: metav1.ConditionTrue, Reason: "X"})
	if err == nil {
		t.Fatal("expected an error when the ServiceUnit doesn't exist, got nil")
	}
}

func TestMergeCondition(t *testing.T) {
	existing := []metav1.Condition{
		{Type: "A", Status: metav1.ConditionTrue, Reason: "orig"},
		{Type: "B", Status: metav1.ConditionFalse, Reason: "orig"},
	}

	t.Run("replaces existing type", func(t *testing.T) {
		got := mergeCondition(append([]metav1.Condition(nil), existing...), metav1.Condition{Type: "A", Status: metav1.ConditionFalse, Reason: "updated"})
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Reason != "updated" || got[0].Status != metav1.ConditionFalse {
			t.Errorf("A not replaced in place: %+v", got[0])
		}
	})

	t.Run("appends new type", func(t *testing.T) {
		got := mergeCondition(append([]metav1.Condition(nil), existing...), metav1.Condition{Type: "C", Status: metav1.ConditionTrue, Reason: "new"})
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		if got[2].Type != "C" {
			t.Errorf("C not appended: %+v", got)
		}
	})
}
