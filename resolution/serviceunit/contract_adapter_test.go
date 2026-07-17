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

package serviceunit

import (
	"context"
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

func TestResolveServiceUnit_Stub(t *testing.T) {
	// ResolveServiceUnit is currently a stub (real logic commented out
	// pending the protojson decode path) — this test documents current
	// behavior so a change to the stub is caught rather than silently
	// altering what every caller receives.
	resolved, err := ResolveServiceUnit(nil)
	if resolved != nil || err != nil {
		t.Fatalf("expected (nil, nil) from the stub, got (%+v, %v)", resolved, err)
	}
}

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	resolved, err := a.Resolve(context.Background(), nil)
	if resolved != nil || err != nil {
		t.Fatalf("expected (nil, nil), got (%+v, %v)", resolved, err)
	}
}

func TestToServiceUnitContract_NilSpec(t *testing.T) {
	var s *ResolvedServiceUnitSpec
	if got := s.ToServiceUnitContract(); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToServiceUnitContract_Static(t *testing.T) {
	s := &ResolvedServiceUnitSpec{
		Type:          commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
		Image:         "foo:latest",
		ContainerPort: 8080,
		Size:          1,
		AppType:       "web",
		StackType:     "node",
	}
	out := s.ToServiceUnitContract()
	if out.Image != "foo:latest" || out.ContainerPort != 8080 {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.Type.Type != commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC {
		t.Fatalf("unexpected Type: %v", out.Type.Type)
	}
	if out.BuildRef != nil {
		t.Fatal("expected nil BuildRef for STATIC type")
	}
}

func TestToServiceUnitContract_Build(t *testing.T) {
	s := &ResolvedServiceUnitSpec{
		Type:     commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
		BuildRef: &ResolvedBuildRef{Name: "b1", Namespace: "default"},
	}
	out := s.ToServiceUnitContract()
	if out.BuildRef == nil || out.BuildRef.Name != "b1" || out.BuildRef.Namespace != "default" {
		t.Fatalf("unexpected BuildRef: %+v", out.BuildRef)
	}
	if out.Image != "" {
		t.Fatalf("expected empty Image for BUILD type, got %q", out.Image)
	}
}

func TestToServiceUnitContract_BuildWithNilBuildRef(t *testing.T) {
	s := &ResolvedServiceUnitSpec{
		Type:     commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
		BuildRef: nil,
	}
	out := s.ToServiceUnitContract()
	if out.BuildRef != nil {
		t.Fatalf("expected nil BuildRef when resolved BuildRef is nil, got %+v", out.BuildRef)
	}
}

func TestToServiceUnitContract_UnknownType(t *testing.T) {
	s := &ResolvedServiceUnitSpec{Type: commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_UNSPECIFIED}
	out := s.ToServiceUnitContract()
	if out.Image != "" || out.BuildRef != nil {
		t.Fatalf("expected no type-specific fields set, got %+v", out)
	}
}
