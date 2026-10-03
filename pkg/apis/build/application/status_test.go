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
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	buildv1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/build/domain"
)

func newStatusTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := buildv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return scheme
}

func newTestBuild(name string) *buildv1.Build {
	return &buildv1.Build{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
	}
}

func TestStatusWriter_Write_PersistsConditions(t *testing.T) {
	scheme := newStatusTestScheme(t)
	build := newTestBuild("b1")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(build).WithStatusSubresource(build).Build()
	w := NewStatusWriter(c, logr.Discard())

	err := w.Write(context.Background(), build, metav1.Condition{
		Type: "BuildSuccess", Status: metav1.ConditionTrue, Reason: "BuildSucceeded", Message: "ok",
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := &buildv1.Build{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "b1", Namespace: "default"}, got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Status.Conditions) != 1 || got.Status.Conditions[0].Type != "BuildSuccess" {
		t.Fatalf("unexpected conditions: %+v", got.Status.Conditions)
	}
}

// TestStatusWriter_Write_PersistsContract locks in the fix: buildrun-observer
// sets Status.Contract on its local Build object before calling Write and
// expects it to persist (its own package doc says so) — before this fix it
// silently never did, leaving build-observer's applyRetry permanently blind
// to Triggered/Success/ExecutionRef in production.
func TestStatusWriter_Write_PersistsContract(t *testing.T) {
	scheme := newStatusTestScheme(t)
	build := newTestBuild("b2")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(build).WithStatusSubresource(build).Build()
	w := NewStatusWriter(c, logr.Discard())

	status := domain.BuildStatus{Triggered: true, Success: true, ExecutionRef: "run-1", BuildHash: "abc123"}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	build.Status.Contract = runtime.RawExtension{Raw: raw}

	if err := w.Write(context.Background(), build, metav1.Condition{Type: "BuildSuccess", Status: metav1.ConditionTrue}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := &buildv1.Build{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "b2", Namespace: "default"}, got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Status.Contract.Raw) == 0 {
		t.Fatal("expected Status.Contract to be persisted, got empty")
	}
	var gotStatus domain.BuildStatus
	if err := json.Unmarshal(got.Status.Contract.Raw, &gotStatus); err != nil {
		t.Fatalf("unmarshal persisted contract: %v", err)
	}
	if gotStatus != status {
		t.Fatalf("persisted contract = %+v, want %+v", gotStatus, status)
	}
}

// TestStatusWriter_Write_LeavesContractUntouchedWhenCallerDoesNotSetIt
// confirms the primary BuildService.Reconcile path (which never populates
// Status.Contract locally) sees no behavior change: an existing persisted
// Contract is left alone rather than being wiped by a Write call that only
// carries conditions.
func TestStatusWriter_Write_LeavesContractUntouchedWhenCallerDoesNotSetIt(t *testing.T) {
	scheme := newStatusTestScheme(t)
	build := newTestBuild("b3")
	existing := domain.BuildStatus{Triggered: true, Success: true, ExecutionRef: "run-preexisting"}
	raw, err := json.Marshal(existing)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	build.Status.Contract = runtime.RawExtension{Raw: raw}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(build).WithStatusSubresource(build).Build()
	w := NewStatusWriter(c, logr.Discard())

	// A fresh, empty-Contract Build object — as BuildService.Reconcile
	// constructs it — used only as the lookup key.
	caller := &buildv1.Build{ObjectMeta: metav1.ObjectMeta{Name: "b3", Namespace: "default"}}
	if err := w.Write(context.Background(), caller, metav1.Condition{Type: "BuildPending", Status: metav1.ConditionFalse}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := &buildv1.Build{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "b3", Namespace: "default"}, got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	var gotStatus domain.BuildStatus
	if err := json.Unmarshal(got.Status.Contract.Raw, &gotStatus); err != nil {
		t.Fatalf("unmarshal persisted contract: %v", err)
	}
	if gotStatus != existing {
		t.Fatalf("existing contract was disturbed: got %+v, want %+v", gotStatus, existing)
	}
}
