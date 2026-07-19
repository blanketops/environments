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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
)

func newStatusTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return scheme
}

func TestStatusWriter_WriteDeploymentResult(t *testing.T) {
	tests := []struct {
		name       string
		result     *domain.DeploymentResult
		runErr     error
		wantWrite  bool
		wantType   string
		wantStatus metav1.ConditionStatus
	}{
		{
			name:       "run error yields a False Ready condition",
			result:     nil,
			runErr:     errors.New("boom"),
			wantWrite:  true,
			wantType:   "Ready",
			wantStatus: metav1.ConditionFalse,
		},
		{
			name:       "ready phase yields a True Ready condition",
			result:     &domain.DeploymentResult{Phase: domain.DeploymentPhase("Ready"), Message: "all good"},
			runErr:     nil,
			wantWrite:  true,
			wantType:   "Ready",
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:       "other phase yields a True Reconciling condition",
			result:     &domain.DeploymentResult{Phase: domain.DeploymentPhase("Deploying"), Message: "in progress"},
			runErr:     nil,
			wantWrite:  true,
			wantType:   "Reconciling",
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:      "nil result and nil error writes nothing",
			result:    nil,
			runErr:    nil,
			wantWrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newStatusTestScheme(t)
			depl := &environmentv1alpha1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(depl).
				WithStatusSubresource(&environmentv1alpha1.Deployment{}).
				Build()

			w := NewStatusWriter(c, logr.Discard())
			if err := w.WriteDeploymentResult(context.Background(), depl, tt.result, tt.runErr); err != nil {
				t.Fatalf("WriteDeploymentResult: unexpected error: %v", err)
			}

			var got environmentv1alpha1.Deployment
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(depl), &got); err != nil {
				t.Fatalf("Get: %v", err)
			}

			if !tt.wantWrite {
				if len(got.Status.Conditions) != 0 {
					t.Fatalf("conditions = %+v, want none written", got.Status.Conditions)
				}
				return
			}

			if len(got.Status.Conditions) != 1 {
				t.Fatalf("len(conditions) = %d, want 1: %+v", len(got.Status.Conditions), got.Status.Conditions)
			}
			cond := got.Status.Conditions[0]
			if cond.Type != tt.wantType {
				t.Errorf("condition.Type = %q, want %q", cond.Type, tt.wantType)
			}
			if cond.Status != tt.wantStatus {
				t.Errorf("condition.Status = %q, want %q", cond.Status, tt.wantStatus)
			}
		})
	}
}

func TestStatusWriter_WriteDeploymentResult_MergesExistingCondition(t *testing.T) {
	scheme := newStatusTestScheme(t)
	depl := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
		Status: environmentv1alpha1.DeploymentStatus{
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Stale", LastTransitionTime: metav1.Now()},
				{Type: "SomeOtherCondition", Status: metav1.ConditionTrue, Reason: "Unrelated", LastTransitionTime: metav1.Now()},
			},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(depl).
		WithStatusSubresource(&environmentv1alpha1.Deployment{}).
		Build()

	w := NewStatusWriter(c, logr.Discard())
	result := &domain.DeploymentResult{Phase: domain.DeploymentPhase("Ready"), Message: "done"}
	if err := w.WriteDeploymentResult(context.Background(), depl, result, nil); err != nil {
		t.Fatalf("WriteDeploymentResult: %v", err)
	}

	var got environmentv1alpha1.Deployment
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(depl), &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Status.Conditions) != 2 {
		t.Fatalf("len(conditions) = %d, want 2 (Ready updated in place, SomeOtherCondition preserved): %+v", len(got.Status.Conditions), got.Status.Conditions)
	}
	for _, cond := range got.Status.Conditions {
		if cond.Type == "Ready" && cond.Status != metav1.ConditionTrue {
			t.Errorf("Ready condition not updated in place: %+v", cond)
		}
	}
}
