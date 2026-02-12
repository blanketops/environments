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

// Info emits a Normal event.
func (er *EventRecorder) Info(obj client.Object, reason, msg string, args ...interface{}) {
	if er == nil || er.recorder == nil {
		return
	}
	er.recorder.Eventf(obj, corev1.EventTypeNormal, reason, msg, args...)
}

// Warn emits a Warning event.
func (er *EventRecorder) Warn(obj client.Object, reason, msg string, args ...interface{}) {
	if er == nil || er.recorder == nil {
		return
	}
	er.recorder.Eventf(obj, corev1.EventTypeWarning, reason, msg, args...)
}

// FromError emits a Warning event for an error.
func (er *EventRecorder) FromError(obj client.Object, reason string, err error) {
	if err == nil {
		return
	}
	er.Warn(obj, reason, fmt.Sprintf("%v", err))
}
