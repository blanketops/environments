package core

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventRecorder wraps controller-runtime's recorder with simplified helpers.
type EventRecorder struct {
	recorder events.EventRecorder
}

// NewEventRecorder returns a wrapped EventRecorder.
func NewEventRecorder(rec events.EventRecorder) *EventRecorder {
	return &EventRecorder{recorder: rec}
}

// Info emits a Normal event.
func (er *EventRecorder) Info(obj client.Object, reason, action, msg string, args ...interface{}) {
	if er == nil || er.recorder == nil {
		return
	}

	er.recorder.Eventf(
		obj,                    // regarding
		nil,                    // related
		corev1.EventTypeNormal, // type
		reason,                 // reason
		action,                 // action
		msg,                    // note (format string)
		args...,                // format args
	)
}

// Warn emits a Warning event.
func (er *EventRecorder) Warn(obj client.Object, reason, action, msg string, args ...interface{}) {
	if er == nil || er.recorder == nil {
		return
	}

	er.recorder.Eventf(
		obj,
		nil,
		corev1.EventTypeWarning,
		reason,
		action,
		msg,
		args...,
	)
}

// FromError emits a Warning event for an error.
func (er *EventRecorder) FromError(obj client.Object, reason, action string, err error) {
	if err == nil {
		return
	}

	er.Warn(obj, reason, action, "%v", err)
}
