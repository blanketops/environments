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
limitations under the License.n
*/

package application

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/gitrepository/domain"
)

const (
	ConditionReady = "Ready"
)

// StatusWriter mutates GitRepository status IN-MEMORY only.
// Persistence is owned exclusively by the controller layer.
type StatusWriter struct{}

func NewStatusWriter() *StatusWriter {
	return &StatusWriter{}
}

func (w *StatusWriter) Write(
	_ context.Context, // intentionally unused: no API calls here
	cr *sourcesv1alpha1.GitRepository,
	result domain.Result,
	err error,
) error {

	now := metav1.NewTime(time.Now())

	cond := metav1.Condition{
		Type:               ConditionReady,
		ObservedGeneration: cr.Generation,
		LastTransitionTime: now,
	}

	switch result.State {

	case domain.StateReady:
		cond.Status = metav1.ConditionTrue
		cond.Reason = "Ready"
		cond.Message = result.Reason

	case domain.StatePending:
		cond.Status = metav1.ConditionFalse
		cond.Reason = "Pending"
		cond.Message = result.Reason

	case domain.StateError:
		cond.Status = metav1.ConditionFalse
		cond.Reason = "Error"
		cond.Message = result.Reason
	}

	// Hard errors override domain message (controller / provider failures)
	if err != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "Error"
		cond.Message = err.Error()
	}

	// Single authoritative condition mutation
	meta.SetStatusCondition(&cr.Status.Conditions, cond)

	return nil
}
