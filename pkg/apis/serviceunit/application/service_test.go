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

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/serviceunit/domain"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := environmentv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return s
}

// newTestClient builds a fake client seeded with objs. WithStatusSubresource
// is required (not optional) for this controller-runtime version — without
// it, Status().Update() fails confusingly with a "not found" error instead
// of a clear "status subresource not registered" one.
func newTestClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithStatusSubresource(&environmentv1alpha1.ServiceUnit{}).
		WithObjects(objs...).
		Build()
}

func TestDeriveResult(t *testing.T) {
	tests := []struct {
		name string
		unit domain.ServiceUnit
		want domain.Phase
	}{
		{"static with image", domain.ServiceUnit{Type: domain.TypeStatic, Image: "img:v1"}, domain.PhaseReady},
		{"static missing image", domain.ServiceUnit{Type: domain.TypeStatic}, domain.PhaseFailed},
		{"build awaiting image", domain.ServiceUnit{Type: domain.TypeBuild}, domain.PhasePending},
		{"build with image", domain.ServiceUnit{Type: domain.TypeBuild, Image: "img:v1"}, domain.PhaseReady},
		{"supplychain", domain.ServiceUnit{Type: domain.TypeSupplyChain}, domain.PhasePending},
		{"unknown type", domain.ServiceUnit{Type: "bogus"}, domain.PhaseFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveResult(tt.unit)
			if result.Phase != tt.want {
				t.Errorf("Phase = %v, want %v (message: %q)", result.Phase, tt.want, result.Message)
			}
		})
	}
}

func TestServiceUnitConditions(t *testing.T) {
	tests := []struct {
		name       string
		result     domain.Result
		wantType   string
		wantStatus metav1.ConditionStatus
	}{
		{"ready", domain.Result{Phase: domain.PhaseReady}, "ServiceUnitReady", metav1.ConditionTrue},
		{"failed", domain.Result{Phase: domain.PhaseFailed, Message: "boom"}, "ServiceUnitFailed", metav1.ConditionFalse},
		{"pending", domain.Result{Phase: domain.PhasePending, Message: "waiting"}, "ServiceUnitPending", metav1.ConditionFalse},
		{"deploying falls to default", domain.Result{Phase: domain.PhaseDeploying}, "ServiceUnitPending", metav1.ConditionFalse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conds := serviceUnitConditions(tt.result)
			if len(conds) != 1 {
				t.Fatalf("len(conditions) = %d, want 1", len(conds))
			}
			if conds[0].Type != tt.wantType {
				t.Errorf("Type = %q, want %q", conds[0].Type, tt.wantType)
			}
			if conds[0].Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", conds[0].Status, tt.wantStatus)
			}
		})
	}
}

func TestServiceUnitService_Reconcile_NilResolved(t *testing.T) {
	svc := NewServiceUnitService(NewMapper(), NewStatusWriter(nil, logr.Discard()))

	err := svc.Reconcile(context.Background(), nil)
	if !errors.Is(err, domain.ErrServiceUnitNil) {
		t.Fatalf("Reconcile(nil) = %v, want ErrServiceUnitNil", err)
	}
}

func TestServiceUnitService_Reconcile_WritesReadyCondition(t *testing.T) {
	su := &environmentv1alpha1.ServiceUnit{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
	}
	c := newTestClient(t, su)
	svc := NewServiceUnitService(NewMapper(), NewStatusWriter(c, logr.Discard()))

	resolved := &serviceunitResolution.ResolvedServiceUnit{
		ServiceUnit: su,
		Spec: &serviceunitResolution.ResolvedServiceUnitSpec{
			Type:  commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
			Image: "docker.io/blanketops/api:v1.2.3",
		},
	}

	if err := svc.Reconcile(context.Background(), resolved); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var got environmentv1alpha1.ServiceUnit
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(su), &got); err != nil {
		t.Fatalf("Get after Reconcile: %v", err)
	}

	found := false
	for _, cond := range got.Status.Conditions {
		if cond.Type == "ServiceUnitReady" {
			found = true
			if cond.Status != metav1.ConditionTrue {
				t.Errorf("ServiceUnitReady status = %q, want True", cond.Status)
			}
		}
	}
	if !found {
		t.Fatalf("ServiceUnitReady condition not found in %+v", got.Status.Conditions)
	}
}
