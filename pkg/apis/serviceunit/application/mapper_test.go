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
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/serviceunit/domain"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

func newResolvedServiceUnit(name, namespace string, spec *serviceunitResolution.ResolvedServiceUnitSpec) *serviceunitResolution.ResolvedServiceUnit {
	return &serviceunitResolution.ResolvedServiceUnit{
		ServiceUnit: &environmentv1alpha1.ServiceUnit{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		},
		Spec: spec,
	}
}

func TestMapper_MapResolvedToDomain_Static(t *testing.T) {
	rsu := newResolvedServiceUnit("api", "default", &serviceunitResolution.ResolvedServiceUnitSpec{
		Type:          commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
		Image:         "docker.io/blanketops/api:v1.2.3",
		ContainerPort: 8080,
		Size:          3,
		AppType:       "web",
		StackType:     "nodejs",
		RouteRef:      &serviceunitResolution.ResolvedRouteRef{Name: "api-route"},
	})

	su := Mapper{}.MapResolvedToDomain(rsu)

	if su.Name != "api" || su.Namespace != "default" {
		t.Errorf("Name/Namespace = %q/%q, want api/default", su.Name, su.Namespace)
	}
	if su.Type != domain.TypeStatic {
		t.Errorf("Type = %v, want TypeStatic", su.Type)
	}
	if su.Image != "docker.io/blanketops/api:v1.2.3" {
		t.Errorf("Image = %q", su.Image)
	}
	if su.ContainerPort != 8080 || su.Size != 3 {
		t.Errorf("ContainerPort/Size = %d/%d, want 8080/3", su.ContainerPort, su.Size)
	}
	if su.AppType != "web" || su.StackType != "nodejs" {
		t.Errorf("AppType/StackType = %q/%q", su.AppType, su.StackType)
	}
	if su.BuildRef != nil {
		t.Errorf("BuildRef = %+v, want nil for STATIC", su.BuildRef)
	}
	if su.RouteRef == nil || su.RouteRef.Name != "api-route" {
		t.Errorf("RouteRef = %+v, want name=api-route", su.RouteRef)
	}
}

func TestMapper_MapResolvedToDomain_Build(t *testing.T) {
	rsu := newResolvedServiceUnit("worker", "ci", &serviceunitResolution.ResolvedServiceUnitSpec{
		Type:     commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
		BuildRef: &serviceunitResolution.ResolvedBuildRef{Name: "worker-build", Namespace: "ci"},
	})

	su := Mapper{}.MapResolvedToDomain(rsu)

	if su.Type != domain.TypeBuild {
		t.Errorf("Type = %v, want TypeBuild", su.Type)
	}
	if su.BuildRef == nil || su.BuildRef.Name != "worker-build" || su.BuildRef.Namespace != "ci" {
		t.Errorf("BuildRef = %+v, want name=worker-build namespace=ci", su.BuildRef)
	}
	if su.Image != "" {
		t.Errorf("Image = %q, want empty for BUILD", su.Image)
	}
}

func TestMapper_MapResolvedToDomain_SupplyChain(t *testing.T) {
	rsu := newResolvedServiceUnit("chain", "default", &serviceunitResolution.ResolvedServiceUnitSpec{
		Type: commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_SUPPLYCHAIN,
	})

	su := Mapper{}.MapResolvedToDomain(rsu)

	if su.Type != domain.TypeSupplyChain {
		t.Errorf("Type = %v, want TypeSupplyChain", su.Type)
	}
}

func TestMapper_MapResolvedToDomain_UnknownType_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unrecognised type, got none")
		}
	}()

	rsu := newResolvedServiceUnit("bad", "default", &serviceunitResolution.ResolvedServiceUnitSpec{
		Type: commoncontractv1.ServiceUnitType_ServiceUnitType(99),
	})

	Mapper{}.MapResolvedToDomain(rsu)
}
