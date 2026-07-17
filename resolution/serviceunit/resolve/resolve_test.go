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

package resolve

import (
	"testing"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newServiceUnit(t *testing.T, name string, contract string) *environmentv1alpha1.ServiceUnit {
	t.Helper()
	return &environmentv1alpha1.ServiceUnit{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: environmentv1alpha1.ServiceUnitSpec{
			Contract: runtime.RawExtension{Raw: []byte(contract)},
		},
	}
}

func TestResolveServiceUnit_Static(t *testing.T) {
	su := newServiceUnit(t, "api", `{
		"type": "static",
		"image": "docker.io/blanketops/api:v1.2.3",
		"containerPort": 8080,
		"size": 3,
		"appType": "web",
		"stackType": "nodejs",
		"route": {"name": "api-route"}
	}`)

	resolved, err := ResolveServiceUnit(su)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Spec.Type != commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC {
		t.Errorf("Type = %v, want STATIC", resolved.Spec.Type)
	}
	if resolved.Spec.Image != "docker.io/blanketops/api:v1.2.3" {
		t.Errorf("Image = %q, want image ref", resolved.Spec.Image)
	}
	if resolved.Spec.ContainerPort != 8080 {
		t.Errorf("ContainerPort = %d, want 8080", resolved.Spec.ContainerPort)
	}
	if resolved.Spec.Size != 3 {
		t.Errorf("Size = %d, want 3", resolved.Spec.Size)
	}
	if resolved.Spec.AppType != "web" || resolved.Spec.StackType != "nodejs" {
		t.Errorf("AppType/StackType = %q/%q, want web/nodejs", resolved.Spec.AppType, resolved.Spec.StackType)
	}
	if resolved.Spec.RouteRef == nil || resolved.Spec.RouteRef.Name != "api-route" {
		t.Errorf("RouteRef = %+v, want name=api-route", resolved.Spec.RouteRef)
	}
	if resolved.Spec.BuildRef != nil {
		t.Errorf("BuildRef = %+v, want nil for STATIC", resolved.Spec.BuildRef)
	}
}

func TestResolveServiceUnit_Build(t *testing.T) {
	su := newServiceUnit(t, "worker", `{
		"type": "build",
		"buildRef": {"name": "worker-build", "namespace": "ci"},
		"containerPort": 9090,
		"size": 1
	}`)

	resolved, err := ResolveServiceUnit(su)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Spec.Type != commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD {
		t.Errorf("Type = %v, want BUILD", resolved.Spec.Type)
	}
	if resolved.Spec.Image != "" {
		t.Errorf("Image = %q, want empty for BUILD (injection out of scope)", resolved.Spec.Image)
	}
	if resolved.Spec.BuildRef == nil || resolved.Spec.BuildRef.Name != "worker-build" || resolved.Spec.BuildRef.Namespace != "ci" {
		t.Errorf("BuildRef = %+v, want name=worker-build namespace=ci", resolved.Spec.BuildRef)
	}
}

func TestResolveServiceUnit_Errors(t *testing.T) {
	tests := []struct {
		name string
		su   *environmentv1alpha1.ServiceUnit
	}{
		{"nil serviceunit", nil},
		{"empty contract", newServiceUnit(t, "x", "")},
		{"missing type", newServiceUnit(t, "x", `{}`)},
		{"unknown type", newServiceUnit(t, "x", `{"type": "wasm"}`)},
		{"static missing image", newServiceUnit(t, "x", `{"type": "static"}`)},
		{"build missing buildRef", newServiceUnit(t, "x", `{"type": "build"}`)},
		{"build missing buildRef.name", newServiceUnit(t, "x", `{"type": "build", "buildRef": {"namespace": "ci"}}`)},
		{"route missing name", newServiceUnit(t, "x", `{"type": "static", "image": "img:v1", "route": {}}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveServiceUnit(tt.su)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
