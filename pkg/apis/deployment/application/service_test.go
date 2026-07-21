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
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"

	"github.com/blanketops/environments/pkg/apis/deployment/api"
	"github.com/blanketops/environments/pkg/apis/deployment/reconcile"
	"github.com/blanketops/environments/pkg/apis/deployment/strategy"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

func newServiceTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		environmentv1alpha1.AddToScheme,
		appsv1.AddToScheme,
		corev1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			t.Fatalf("AddToScheme: %v", err)
		}
	}
	return scheme
}

func newTestDeploymentService(t *testing.T, c client.Client, scheme *runtime.Scheme) *DeploymentService {
	t.Helper()
	log := logr.Discard()
	return NewDeploymentService(
		intent.NewIntentBuilder(),
		NewStatusWriter(c, log),
		reconcile.NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, log, nil),
			api.NewKustomizeStrategyProvider(c, scheme, log),
			log,
		),
		log,
	)
}

func TestDeploymentService_Reconcile(t *testing.T) {
	scheme := newServiceTestScheme(t)
	depl := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default", Generation: 1},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(depl).
		WithStatusSubresource(&environmentv1alpha1.Deployment{}).
		Build()

	svc := newTestDeploymentService(t, c, scheme)

	resolved := &deploymentResolution.ResolvedDeployment{
		Deployment: depl,
		Spec: &deploymentResolution.ResolvedDeploymentSpec{
			ServiceUnits:           []string{"api"},
			Runtime:                deploymentResolution.RuntimeKubernetes,
			Strategy:               deploymentResolution.StrategyRolling,
			ReconciliationStrategy: deploymentResolution.ReconciliationImperative,
		},
	}
	serviceUnits := []serviceunitResolution.ResolvedServiceUnit{
		{
			ServiceUnit: &environmentv1alpha1.ServiceUnit{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}},
			Spec: &serviceunitResolution.ResolvedServiceUnitSpec{
				Type:          commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
				Image:         "docker.io/blanketops/api:v1",
				ContainerPort: 8080,
				Size:          2,
			},
		},
	}

	if err := svc.Reconcile(context.Background(), resolved, serviceUnits, logr.Discard()); err != nil {
		t.Fatalf("Reconcile: unexpected error: %v", err)
	}

	var gotDeploy appsv1.Deployment
	if err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotDeploy); err != nil {
		t.Fatalf("expected a Deployment to have been applied: %v", err)
	}
	if got, want := gotDeploy.Spec.Template.Spec.Containers[0].Image, "docker.io/blanketops/api:v1"; got != want {
		t.Errorf("applied image = %q, want %q", got, want)
	}

	var gotSvc corev1.Service
	if err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotSvc); err != nil {
		t.Fatalf("expected a Service to have been applied: %v", err)
	}

	var gotCR environmentv1alpha1.Deployment
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(depl), &gotCR); err != nil {
		t.Fatalf("Get Deployment CR: %v", err)
	}
	if len(gotCR.Status.Conditions) == 0 {
		t.Error("expected the Deployment CR to have at least one condition written")
	}
}

func TestDeploymentService_Reconcile_NilResolvedDeploymentSurfacesIntentBuilderError(t *testing.T) {
	scheme := newServiceTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	svc := newTestDeploymentService(t, c, scheme)

	err := svc.Reconcile(context.Background(), &deploymentResolution.ResolvedDeployment{}, nil, logr.Discard())
	if err == nil {
		t.Fatal("expected an error when the ResolvedDeployment has no Spec, got nil")
	}
}
