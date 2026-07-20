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

package reconcile

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	"github.com/blanketops/environments/pkg/apis/deployment/strategy"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(environment): %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(apps): %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(core): %v", err)
	}
	return scheme
}

func TestReconciliationExecutor_Execute(t *testing.T) {
	sourceCR := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
	}

	t.Run("nil intent returns an error", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		_, err := exec.Execute(context.Background(), sourceCR, nil)
		if err == nil {
			t.Fatal("expected an error for nil intent, got nil")
		}
	})

	t.Run("imperative mode dispatches to the runtime provider and applies real objects", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:      "web",
			Namespace: "default",
			Runtime:   intent.RuntimeKubernetes,
			Strategy:  intent.StrategyRolling,
			ServiceUnits: []serviceunitIntent.ServiceUnitIntent{
				{Name: "api", Image: "docker.io/blanketops/api:v1", Port: 8080, Size: 1},
			},
			// ManifestsRepo left nil — this is the imperative path.
		}

		result, err := exec.Execute(context.Background(), sourceCR, dIntent)
		if err != nil {
			t.Fatalf("Execute: unexpected error: %v", err)
		}
		if result == nil || len(result.ServiceUnits) != 1 {
			t.Fatalf("result = %+v, want one ServiceUnitResult", result)
		}
		if result.ServiceUnits[0].Phase == domain.ServiceUnitPhase("Failed") {
			t.Errorf("ServiceUnit reported Failed: %+v", result.ServiceUnits[0])
		}
	})

	t.Run("gitops mode with helm strategy is rejected as not implemented", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil, // Kustomizer unused on this branch — Helm returns before touching it.
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:                   "web",
			Namespace:              "default",
			ReconciliationStrategy: intent.ReconciliationHelm,
			ManifestsRepo:          &intent.ManifestsRepo{URL: "https://example.invalid/repo.git"},
		}

		_, err := exec.Execute(context.Background(), sourceCR, dIntent)
		if err == nil {
			t.Fatal("expected an error for helm reconciliation, got nil")
		}
	})

	t.Run("gitops mode with an unsupported strategy is rejected", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:                   "web",
			Namespace:              "default",
			ReconciliationStrategy: intent.ReconciliationStrategy("Unknown"),
			ManifestsRepo:          &intent.ManifestsRepo{URL: "https://example.invalid/repo.git"},
		}

		_, err := exec.Execute(context.Background(), sourceCR, dIntent)
		if err == nil {
			t.Fatal("expected an error for an unsupported reconciliation strategy, got nil")
		}
	})
}

func TestReconciliationExecutor_Teardown(t *testing.T) {
	sourceCR := &environmentv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
	}

	t.Run("nil intent returns an error", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		if err := exec.Teardown(context.Background(), sourceCR, nil); err == nil {
			t.Fatal("expected an error for nil intent, got nil")
		}
	})

	t.Run("imperative mode dispatches to the runtime provider and deletes real objects", func(t *testing.T) {
		scheme := newTestScheme(t)
		existingDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}}
		existingSvc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingDeploy, existingSvc).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:      "web",
			Namespace: "default",
			Runtime:   intent.RuntimeKubernetes,
			Strategy:  intent.StrategyRolling,
			ServiceUnits: []serviceunitIntent.ServiceUnitIntent{
				{Name: "api", Image: "docker.io/blanketops/api:v1", Port: 8080, Size: 1},
			},
			// ManifestsRepo left nil — this is the imperative path.
		}

		if err := exec.Teardown(context.Background(), sourceCR, dIntent); err != nil {
			t.Fatalf("Teardown: unexpected error: %v", err)
		}

		var gotDeploy appsv1.Deployment
		err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotDeploy)
		if !apierrors.IsNotFound(err) {
			t.Errorf("expected the Deployment to have been deleted, Get err = %v", err)
		}
	})

	t.Run("gitops mode with helm strategy is rejected as not implemented", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil, // Kustomizer unused on this branch — Helm returns before touching it.
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:                   "web",
			Namespace:              "default",
			ReconciliationStrategy: intent.ReconciliationHelm,
			ManifestsRepo:          &intent.ManifestsRepo{URL: "https://example.invalid/repo.git"},
		}

		if err := exec.Teardown(context.Background(), sourceCR, dIntent); err == nil {
			t.Fatal("expected an error for helm reconciliation, got nil")
		}
	})

	t.Run("gitops mode with an unsupported strategy is rejected", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil,
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:                   "web",
			Namespace:              "default",
			ReconciliationStrategy: intent.ReconciliationStrategy("Unknown"),
			ManifestsRepo:          &intent.ManifestsRepo{URL: "https://example.invalid/repo.git"},
		}

		if err := exec.Teardown(context.Background(), sourceCR, dIntent); err == nil {
			t.Fatal("expected an error for an unsupported reconciliation strategy, got nil")
		}
	})

	t.Run("gitops kustomize mode requires the environment-type label", func(t *testing.T) {
		scheme := newTestScheme(t)
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		exec := NewReconciliationExecutor(
			strategy.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
			nil, // Kustomizer unused — missing label returns before touching it.
			logr.Discard(),
		)

		dIntent := &intent.DeploymentIntent{
			Name:                   "web",
			Namespace:              "default",
			ReconciliationStrategy: intent.ReconciliationKustomize,
			ManifestsRepo:          &intent.ManifestsRepo{URL: "https://example.invalid/repo.git"},
		}

		// sourceCR has no environments.blanketops.dev/type label.
		if err := exec.Teardown(context.Background(), sourceCR, dIntent); err == nil {
			t.Fatal("expected an error for a missing environment-type label, got nil")
		}
	})
}
