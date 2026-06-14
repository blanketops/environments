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

This file owns event recording. Kubernetes events are the primary observability
surface for reconciliation progress — they appear in kubectl describe output and
are consumed by monitoring tooling. Every domain stage that succeeds or fails
emits an event via EventRecorder.

EventRecorder abstracts over two client-go recorder APIs:

  - recordv1 (client-go/tools/record): the legacy recorder, returned by
    mgr.GetEventRecorder(). Used by most controller-runtime setups today.
  - eventsv1 (client-go/tools/events): the structured recorder backed by the
    v1 Events API. More expressive but less commonly wired at setup time.

Both are supported transparently — the caller uses the same Normal/Warn/Info
surface regardless of which recorder was injected. A nil or empty recorder is
always safe; all methods guard against nil receivers and nil objects.
*/
package core

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/client-go/tools/events"
	recordv1 "k8s.io/client-go/tools/record"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventRecorder wraps both client-go recorder APIs behind a unified interface.
// Domains call Normal, Warn, Info, FromError — never the underlying recorders
// directly. This keeps domain code decoupled from the recorder implementation
// and safe against nil recorders at setup time.
type EventRecorder struct {
	// recordRecorder is the legacy client-go/tools/record recorder.
	// Populated when NewEventRecorder receives a recordv1.EventRecorder.
	recordRecorder recordv1.EventRecorder
	// eventsRecorder is the structured client-go/tools/events recorder.
	// Populated when NewEventRecorder receives an eventsv1.EventRecorder.
	eventsRecorder eventsv1.EventRecorder
}

// NewEventRecorder constructs an EventRecorder from either recorder type.
// Accepts recordv1.EventRecorder or eventsv1.EventRecorder — the concrete
// type is detected at construction time and stored in the appropriate field.
// An unrecognised input returns a no-op recorder so callers are always safe.
func NewEventRecorder(rec interface{}) *EventRecorder {
	switch r := rec.(type) {
	case recordv1.EventRecorder:
		return &EventRecorder{recordRecorder: r}
	case eventsv1.EventRecorder:
		return &EventRecorder{eventsRecorder: r}
	}
	return &EventRecorder{}
}

// Event emits a Kubernetes event on obj with the given eventType, reason, and
// formatted message. Dispatches to whichever underlying recorder is populated.
//
// Safe to call with a nil receiver or nil obj — both are no-ops.
// Domains should prefer Normal, Warn, Info, and FromError over calling
// Event directly.
func (er *EventRecorder) Event(
	obj client.Object,
	eventType string,
	reason string,
	msg string,
	args ...interface{},
) {
	if er == nil || obj == nil {
		return
	}

	message := msg
	if len(args) > 0 {
		message = fmt.Sprintf(msg, args...)
	}

	// ------------------------------------------------
	// Legacy recorder (client-go/tools/record).
	//
	// Prefer Eventf when args are present to avoid a double-format pass —
	// Eventf handles the format string internally.
	// ------------------------------------------------
	if er.recordRecorder != nil {
		if len(args) > 0 {
			er.recordRecorder.Eventf(obj, eventType, reason, msg, args...)
			return
		}
		er.recordRecorder.Event(obj, eventType, reason, message)
		return
	}

	// ------------------------------------------------
	// Structured recorder (client-go/tools/events).
	//
	// Requires a runtime.Object — client.Object satisfies this in practice
	// but the assertion guards against edge cases at the type boundary.
	// The action field mirrors reason; the related object is nil since
	// BlanketOps events are always scoped to the owning CR.
	// ------------------------------------------------
	if er.eventsRecorder != nil {
		runtimeObj, ok := obj.(runtime.Object)
		if !ok {
			return
		}
		er.eventsRecorder.Eventf(
			runtimeObj,
			nil,       // related object — not used
			eventType,
			reason,
			reason,  // action mirrors reason
			message,
		)
	}
}

// Normal emits a Normal-type event on obj. Used for successful stage
// transitions in the reconciliation pipeline.
func (er *EventRecorder) Normal(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeNormal, reason, msg, args...)
}

// Info is an alias for Normal. Used at call sites where "informational"
// reads more naturally than "normal" (e.g. deletion acknowledgements).
func (er *EventRecorder) Info(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Normal(obj, reason, msg, args...)
}

// Warn emits a Warning-type event on obj. Used for failed stage transitions
// and error conditions in the reconciliation pipeline.
func (er *EventRecorder) Warn(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeWarning, reason, msg, args...)
}

// FromError emits a Warning event with the error message as the note.
// No-op when err is nil. The standard call site in domain Handle methods:
//
//	d.events.FromError(buildCR, "BuildResolveFailed", err)
func (er *EventRecorder) FromError(
	obj client.Object,
	reason string,
	err error,
) {
	if err == nil {
		return
	}
	er.Warn(obj, reason, "%v", err)
}

// FromErrorf emits a Warning event combining a context message and the
// error. No-op when err is nil. Use when the error alone lacks enough
// context for an operator reading kubectl describe output.
//
//	d.events.FromErrorf(buildCR, "BuildPrerequisitesFailed", err, "namespace %s", ns)
func (er *EventRecorder) FromErrorf(
	obj client.Object,
	reason string,
	err error,
	msg string,
	args ...interface{},
) {
	if err == nil {
		return
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	er.Warn(obj, reason, "%s: %v", msg, err)
}