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

package api

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

func newK8STestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(apps): %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(core): %v", err)
	}
	return scheme
}

func testDeploymentIntent() *intent.DeploymentIntent {
	return &intent.DeploymentIntent{
		Name:      "web",
		Namespace: "default",
		Runtime:   intent.RuntimeKubernetes,
	}
}

func TestK8SProvider_ApplyServiceUnit(t *testing.T) {
	scheme := newK8STestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	p := NewK8SProvider(c, scheme, logr.Discard(), nil)

	su := &serviceunitIntent.ServiceUnitIntent{Name: "api", Image: "docker.io/blanketops/api:v1", Port: 8080, Size: 2}

	result, err := p.ApplyServiceUnit(context.Background(), testDeploymentIntent(), su)
	if err != nil {
		t.Fatalf("ApplyServiceUnit: unexpected error: %v", err)
	}
	if result.Name != "api" || result.Image != "docker.io/blanketops/api:v1" {
		t.Errorf("result = %+v, want name=api image=docker.io/blanketops/api:v1", result)
	}
	// The fake client has no Deployment controller, so ReadyReplicas never
	// populates — Deploying, not Ready, is the correct observed phase here.
	if result.Phase != domain.ServiceUnitPhase("Deploying") {
		t.Errorf("Phase = %q, want Deploying (fake client never marks pods ready)", result.Phase)
	}

	var gotDeploy appsv1.Deployment
	if err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotDeploy); err != nil {
		t.Fatalf("expected a Deployment to have been applied: %v", err)
	}
	if got, want := *gotDeploy.Spec.Replicas, int32(2); got != want {
		t.Errorf("Replicas = %d, want %d", got, want)
	}
	if got, want := gotDeploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort, int32(8080); got != want {
		t.Errorf("ContainerPort = %d, want %d", got, want)
	}

	var gotSvc corev1.Service
	if err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotSvc); err != nil {
		t.Fatalf("expected a Service to have been applied: %v", err)
	}
	if got, want := gotSvc.Spec.Ports[0].Port, int32(8080); got != want {
		t.Errorf("Service port = %d, want %d", got, want)
	}
}

func TestK8SProvider_IsDeploymentReady(t *testing.T) {
	tests := []struct {
		name    string
		deploy  *appsv1.Deployment
		want    bool
		wantErr bool
	}{
		{
			name: "ready replicas match spec replicas",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
				Status:     appsv1.DeploymentStatus{ReadyReplicas: 3},
			},
			want: true,
		},
		{
			name: "ready replicas below spec replicas",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
				Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
			},
			want: false,
		},
		{
			name: "nil spec replicas is never ready",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newK8STestScheme(t)
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.deploy).
				WithStatusSubresource(&appsv1.Deployment{}).
				Build()
			p := NewK8SProvider(c, scheme, logr.Discard(), nil)

			ready, err := p.isDeploymentReady(context.Background(), testDeploymentIntent(), &serviceunitIntent.ServiceUnitIntent{Name: "api"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("isDeploymentReady: err = %v, wantErr = %v", err, tt.wantErr)
			}
			if ready != tt.want {
				t.Errorf("ready = %v, want %v", ready, tt.want)
			}
		})
	}

	t.Run("missing deployment returns an error", func(t *testing.T) {
		scheme := newK8STestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		p := NewK8SProvider(c, scheme, logr.Discard(), nil)

		if _, err := p.isDeploymentReady(context.Background(), testDeploymentIntent(), &serviceunitIntent.ServiceUnitIntent{Name: "missing"}); err == nil {
			t.Fatal("expected an error for a missing Deployment, got nil")
		}
	})
}

func TestK8SProvider_Teardown(t *testing.T) {
	scheme := newK8STestScheme(t)
	existingDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}}
	existingSvc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingDeploy, existingSvc).Build()
	p := NewK8SProvider(c, scheme, logr.Discard(), nil)

	dIntent := &intent.DeploymentIntent{
		Name:      "web",
		Namespace: "default",
		ServiceUnits: []serviceunitIntent.ServiceUnitIntent{
			{Name: "api"},
		},
	}

	if err := p.Teardown(context.Background(), dIntent); err != nil {
		t.Fatalf("Teardown: unexpected error: %v", err)
	}

	var gotDeploy appsv1.Deployment
	err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotDeploy)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected Deployment to be deleted, Get err = %v", err)
	}

	var gotSvc corev1.Service
	err = c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotSvc)
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected Service to be deleted, Get err = %v", err)
	}

	// Idempotent: tearing down again (nothing left to delete) must not error.
	if err := p.Teardown(context.Background(), dIntent); err != nil {
		t.Fatalf("Teardown (idempotent second call): unexpected error: %v", err)
	}
}
