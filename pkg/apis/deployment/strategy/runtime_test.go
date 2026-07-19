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

package strategy

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

func newRuntimeTestScheme(t *testing.T) *runtime.Scheme {
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

func TestRuntimeProvider_Execute(t *testing.T) {
	tests := []struct {
		name    string
		runtime intent.Runtime
		wantErr bool
	}{
		{"kubernetes dispatches to K8SStrategy", intent.RuntimeKubernetes, false},
		{"knative is not implemented", intent.RuntimeKnative, true},
		{"ecs is not implemented", intent.RuntimeECS, true},
		{"unknown runtime is rejected", intent.Runtime("blanketops.dev/mystery-runtime"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newRuntimeTestScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := NewRuntimeProvider(c, scheme, logr.Discard(), nil)

			dIntent := &intent.DeploymentIntent{
				Name:      "web",
				Namespace: "default",
				Runtime:   tt.runtime,
				Strategy:  intent.StrategyRolling,
				ServiceUnits: []serviceunitIntent.ServiceUnitIntent{
					{Name: "api", Image: "docker.io/blanketops/api:v1", Port: 8080, Size: 1},
				},
			}

			result, err := p.Execute(context.Background(), dIntent)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for runtime %q, got nil", tt.runtime)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute(%q): unexpected error: %v", tt.runtime, err)
			}
			if result == nil {
				t.Fatalf("Execute(%q): result is nil", tt.runtime)
			}

			var gotDeploy appsv1.Deployment
			if err := c.Get(context.Background(), client.ObjectKey{Name: "api", Namespace: "default"}, &gotDeploy); err != nil {
				t.Fatalf("expected the Kubernetes strategy to have applied a Deployment: %v", err)
			}
		})
	}
}

func TestDeriveDeploymentPhase(t *testing.T) {
	tests := []struct {
		name    string
		results []domain.ServiceUnitResult
		want    domain.DeploymentPhase
	}{
		{"no results is pending", nil, domain.DeploymentPhase("Pending")},
		{
			"any failed result fails the deployment",
			[]domain.ServiceUnitResult{
				{Phase: domain.ServiceUnitPhase("Ready")},
				{Phase: domain.ServiceUnitPhase("Failed")},
			},
			domain.DeploymentPhase("Failed"),
		},
		{
			"all ready is ready",
			[]domain.ServiceUnitResult{
				{Phase: domain.ServiceUnitPhase("Ready")},
				{Phase: domain.ServiceUnitPhase("Ready")},
			},
			domain.DeploymentPhase("Ready"),
		},
		{
			"a mix of ready and deploying is still deploying",
			[]domain.ServiceUnitResult{
				{Phase: domain.ServiceUnitPhase("Ready")},
				{Phase: domain.ServiceUnitPhase("Deploying")},
			},
			domain.DeploymentPhase("Deploying"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveDeploymentPhase(tt.results); got != tt.want {
				t.Errorf("deriveDeploymentPhase(%+v) = %q, want %q", tt.results, got, tt.want)
			}
		})
	}
}
