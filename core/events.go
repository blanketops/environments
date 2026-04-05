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

package core

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/client-go/tools/events"
	recordv1 "k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventRecorder struct {
	recordRecorder recordv1.EventRecorder
	eventsRecorder eventsv1.EventRecorder
}

func NewEventRecorder(rec interface{}) *EventRecorder {

	switch r := rec.(type) {

	case recordv1.EventRecorder:
		return &EventRecorder{recordRecorder: r}

	case eventsv1.EventRecorder:
		return &EventRecorder{eventsRecorder: r}
	}

	return &EventRecorder{}
}

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

	//------------------------------------------------
	// OLD recorder (client-go/tools/record)
	//------------------------------------------------

	if er.recordRecorder != nil {

		if len(args) > 0 {
			er.recordRecorder.Eventf(obj, eventType, reason, msg, args...)
			return
		}

		er.recordRecorder.Event(obj, eventType, reason, message)
		return
	}

	//------------------------------------------------
	// NEW structured recorder (client-go/tools/events)
	//------------------------------------------------

	if er.eventsRecorder != nil {

		runtimeObj, ok := obj.(runtime.Object)
		if !ok {
			return
		}

		er.eventsRecorder.Eventf(
			runtimeObj,
			nil,       // related
			eventType, // Normal / Warning
			reason,    // reason
			reason,    // action
			message,   // note
		)
	}
}

func (er *EventRecorder) Normal(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeNormal, reason, msg, args...)
}

func (er *EventRecorder) Info(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Normal(obj, reason, msg, args...)
}

func (er *EventRecorder) Warn(
	obj client.Object,
	reason string,
	msg string,
	args ...interface{},
) {
	er.Event(obj, corev1.EventTypeWarning, reason, msg, args...)
}

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
