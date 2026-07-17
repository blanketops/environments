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

package events

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type recordEventCall struct {
	object            runtime.Object
	eventtype, reason string
	message           string
}

type recordEventfCall struct {
	object            runtime.Object
	eventtype, reason string
	messageFmt        string
	args              []interface{}
}

type fakeRecordRecorder struct {
	eventCalls  []recordEventCall
	eventfCalls []recordEventfCall
}

func (f *fakeRecordRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	f.eventCalls = append(f.eventCalls, recordEventCall{object, eventtype, reason, message})
}

func (f *fakeRecordRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	f.eventfCalls = append(f.eventfCalls, recordEventfCall{object, eventtype, reason, messageFmt, args})
}

func (f *fakeRecordRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}

type eventsCall struct {
	regarding, related runtime.Object
	eventtype, reason  string
	action, note       string
	args               []interface{}
}

type fakeEventsRecorder struct {
	calls []eventsCall
}

func (f *fakeEventsRecorder) Eventf(regarding, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
	f.calls = append(f.calls, eventsCall{regarding, related, eventtype, reason, action, note, args})
}

func testPod() *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}}
}

func TestNewEventRecorder_RecordV1(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	if er.recordRecorder == nil {
		t.Fatal("expected recordRecorder to be set")
	}
	if er.eventsRecorder != nil {
		t.Fatal("expected eventsRecorder to remain nil")
	}
}

func TestNewEventRecorder_EventsV1(t *testing.T) {
	rec := &fakeEventsRecorder{}
	er := NewEventRecorder(rec)
	if er.eventsRecorder == nil {
		t.Fatal("expected eventsRecorder to be set")
	}
	if er.recordRecorder != nil {
		t.Fatal("expected recordRecorder to remain nil")
	}
}

func TestNewEventRecorder_UnrecognisedType(t *testing.T) {
	er := NewEventRecorder("not a recorder")
	if er.recordRecorder != nil || er.eventsRecorder != nil {
		t.Fatal("expected both recorder fields to remain nil for an unrecognised input")
	}
}

func TestEventRecorder_Event_NilReceiver(t *testing.T) {
	var er *EventRecorder
	er.Event(testPod(), corev1.EventTypeNormal, "Reason", "message") // must not panic
}

func TestEventRecorder_Event_NilObj(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.Event(nil, corev1.EventTypeNormal, "Reason", "message")
	if len(rec.eventCalls) != 0 || len(rec.eventfCalls) != 0 {
		t.Fatal("expected no calls to the underlying recorder for a nil obj")
	}
}

func TestEventRecorder_Event_RecordRecorder_NoArgs(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)

	er.Event(testPod(), corev1.EventTypeNormal, "Reason", "plain message")

	if len(rec.eventCalls) != 1 {
		t.Fatalf("expected exactly one Event call, got %d", len(rec.eventCalls))
	}
	if len(rec.eventfCalls) != 0 {
		t.Fatalf("expected Eventf not to be called when no args are given, got %d calls", len(rec.eventfCalls))
	}
	got := rec.eventCalls[0]
	if got.eventtype != corev1.EventTypeNormal || got.reason != "Reason" || got.message != "plain message" {
		t.Fatalf("unexpected Event call: %+v", got)
	}
}

func TestEventRecorder_Event_RecordRecorder_WithArgs(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)

	er.Event(testPod(), corev1.EventTypeWarning, "Reason", "failed: %s", "boom")

	if len(rec.eventfCalls) != 1 {
		t.Fatalf("expected exactly one Eventf call, got %d", len(rec.eventfCalls))
	}
	if len(rec.eventCalls) != 0 {
		t.Fatalf("expected Event not to be called when args are given, got %d calls", len(rec.eventCalls))
	}
	got := rec.eventfCalls[0]
	if got.eventtype != corev1.EventTypeWarning || got.reason != "Reason" || got.messageFmt != "failed: %s" {
		t.Fatalf("unexpected Eventf call: %+v", got)
	}
	if len(got.args) != 1 || got.args[0] != "boom" {
		t.Fatalf("unexpected Eventf args: %+v", got.args)
	}
}

func TestEventRecorder_Event_EventsRecorder(t *testing.T) {
	rec := &fakeEventsRecorder{}
	er := NewEventRecorder(rec)

	er.Event(testPod(), corev1.EventTypeNormal, "BuildResolved", "resolved ok")

	if len(rec.calls) != 1 {
		t.Fatalf("expected exactly one Eventf call, got %d", len(rec.calls))
	}
	got := rec.calls[0]
	if got.eventtype != corev1.EventTypeNormal || got.reason != "BuildResolved" {
		t.Fatalf("unexpected call: %+v", got)
	}
	if got.action != got.reason {
		t.Fatalf("expected action to mirror reason, got action=%q reason=%q", got.action, got.reason)
	}
	if got.note != "resolved ok" {
		t.Fatalf("expected note %q, got %q", "resolved ok", got.note)
	}
	if got.related != nil {
		t.Fatalf("expected related to be nil, got %v", got.related)
	}
}

func TestEventRecorder_Normal(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.Normal(testPod(), "Reason", "msg")
	if len(rec.eventCalls) != 1 || rec.eventCalls[0].eventtype != corev1.EventTypeNormal {
		t.Fatalf("expected a Normal-type event, got %+v", rec.eventCalls)
	}
}

func TestEventRecorder_Info_AliasesNormal(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.Info(testPod(), "Reason", "msg")
	if len(rec.eventCalls) != 1 || rec.eventCalls[0].eventtype != corev1.EventTypeNormal {
		t.Fatalf("expected Info to emit a Normal-type event, got %+v", rec.eventCalls)
	}
}

func TestEventRecorder_Warn(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.Warn(testPod(), "Reason", "msg")
	if len(rec.eventCalls) != 1 || rec.eventCalls[0].eventtype != corev1.EventTypeWarning {
		t.Fatalf("expected a Warning-type event, got %+v", rec.eventCalls)
	}
}

func TestEventRecorder_FromError_NilErr(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.FromError(testPod(), "Reason", nil)
	if len(rec.eventCalls) != 0 {
		t.Fatal("expected no event for a nil error")
	}
}

func TestEventRecorder_FromError_NonNilErr(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.FromError(testPod(), "BuildResolveFailed", errors.New("boom"))

	// FromError formats with args ("%v", err), so Event routes through the
	// Eventf branch (len(args) > 0), not the plain Event call.
	if len(rec.eventfCalls) != 1 {
		t.Fatalf("expected exactly one Eventf call, got %d", len(rec.eventfCalls))
	}
	got := rec.eventfCalls[0]
	if got.eventtype != corev1.EventTypeWarning || got.reason != "BuildResolveFailed" {
		t.Fatalf("unexpected event: %+v", got)
	}
	if got.messageFmt != "%v" || len(got.args) != 1 || !strings.Contains(got.args[0].(error).Error(), "boom") {
		t.Fatalf("expected format %%v with the error as the sole arg, got fmt=%q args=%+v", got.messageFmt, got.args)
	}
}

func TestEventRecorder_FromErrorf_NilErr(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.FromErrorf(testPod(), "Reason", nil, "context %s", "extra")
	if len(rec.eventCalls) != 0 {
		t.Fatal("expected no event for a nil error")
	}
}

func TestEventRecorder_FromErrorf_NonNilErr(t *testing.T) {
	rec := &fakeRecordRecorder{}
	er := NewEventRecorder(rec)
	er.FromErrorf(testPod(), "BuildPrerequisitesFailed", errors.New("boom"), "namespace %s", "default")

	// FromErrorf formats with args ("%s: %v", msg, err), so this also routes
	// through the Eventf branch.
	if len(rec.eventfCalls) != 1 {
		t.Fatalf("expected exactly one Eventf call, got %d", len(rec.eventfCalls))
	}
	got := rec.eventfCalls[0]
	if got.eventtype != corev1.EventTypeWarning {
		t.Fatalf("expected a Warning event, got %+v", got)
	}
	if len(got.args) != 2 || got.args[0] != "namespace default" {
		t.Fatalf("expected args [\"namespace default\", err], got %+v", got.args)
	}
}
