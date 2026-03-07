package core

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventRecorder wraps controller-runtime's recorder with simplified helpers.
type EventRecorder struct {
	recorder record.EventRecorder
}

// NewEventRecorder returns a wrapped EventRecorder.
func NewEventRecorder(rec record.EventRecorder) *EventRecorder {
	return &EventRecorder{recorder: rec}
}

// Event emits a Kubernetes event with the provided type.
func (er *EventRecorder) Event(
	obj client.Object,
	eventType string,
	reason string,
	msg string,
	args ...interface{},
) {
	if er == nil || er.recorder == nil || obj == nil {
		return
	}

	er.recorder.Eventf(obj, eventType, reason, msg, args...)
}

// Normal emits a Kubernetes Normal event.
func (er *EventRecorder) Normal(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeNormal, reason, msg, args...)
}

// Info is kept for backward compatibility (alias of Normal).
func (er *EventRecorder) Info(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Normal(obj, reason, msg, args...)
}

// Warn emits a Kubernetes Warning event.
func (er *EventRecorder) Warn(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeWarning, reason, msg, args...)
}

// FromError emits a Warning event derived from an error.
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

// FromErrorf emits a Warning event with formatted context.
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

	full := fmt.Sprintf(msg, args...)
	er.Warn(obj, reason, "%s: %v", full, err)
}
