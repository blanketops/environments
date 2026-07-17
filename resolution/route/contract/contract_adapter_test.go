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

package contract

import (
	"testing"

	commonv1 "github.com/blanketops/environments-contract/blanketops/common/v1"

	"github.com/blanketops/environments/resolution/route/resolve"
)

func TestToRouteContract_NilRoute(t *testing.T) {
	out, err := ToRouteContract(nil)
	if err != nil || out != nil {
		t.Fatalf("expected (nil, nil), got (%+v, %v)", out, err)
	}
}

func TestToRouteContract_NilSpec(t *testing.T) {
	out, err := ToRouteContract(&resolve.ResolvedRoute{})
	if err != nil || out != nil {
		t.Fatalf("expected (nil, nil), got (%+v, %v)", out, err)
	}
}

func TestToRouteContract_Populated(t *testing.T) {
	rt := resolve.RuntimeKubernetesContainer
	r := &resolve.ResolvedRoute{
		Spec: &resolve.ResolvedRouteSpec{
			Host:           "x",
			Enabled:        true,
			Path:           "/",
			TLSEnabled:     true,
			Runtime:        &rt,
			ServiceUnitRef: &resolve.ResolvedServiceUnitRef{Name: "su1"},
		},
	}
	out, err := ToRouteContract(r)
	if err != nil {
		t.Fatalf("ToRouteContract: %v", err)
	}
	if out.Host != "x" || !out.Enabled || !out.TlsEnabled {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.Runtime.Runtime != commonv1.RouteRuntime_ROUTE_RUNTIME_KUBERNETES_CONTAINER {
		t.Fatalf("unexpected runtime: %v", out.Runtime.Runtime)
	}
	if out.ServiceUnitRef.Name != "su1" {
		t.Fatalf("unexpected ServiceUnitRef: %+v", out.ServiceUnitRef)
	}
}

func TestMapRuntime(t *testing.T) {
	knative := resolve.RuntimeKnativeService
	k8s := resolve.RuntimeKubernetesContainer
	gw := resolve.RuntimeGatewayAPI
	unknown := resolve.Runtime("bogus")

	cases := []struct {
		name string
		rt   *resolve.Runtime
		want commonv1.RouteRuntime_RouteRuntime
	}{
		{"nil", nil, commonv1.RouteRuntime_ROUTE_RUNTIME_UNSPECIFIED},
		{"knative", &knative, commonv1.RouteRuntime_ROUTE_RUNTIME_KNATIVE_SERVICE},
		{"kubernetes", &k8s, commonv1.RouteRuntime_ROUTE_RUNTIME_KUBERNETES_CONTAINER},
		{"gateway", &gw, commonv1.RouteRuntime_ROUTE_RUNTIME_GATEWAY_API},
		{"unknown", &unknown, commonv1.RouteRuntime_ROUTE_RUNTIME_UNSPECIFIED},
	}
	for _, tc := range cases {
		if got := mapRuntime(tc.rt); got.Runtime != tc.want {
			t.Fatalf("%s: mapRuntime = %v, want %v", tc.name, got.Runtime, tc.want)
		}
	}
}

func TestMapServiceUnitRef(t *testing.T) {
	if got := mapServiceUnitRef(nil); got != nil {
		t.Fatalf("expected nil for nil ref, got %+v", got)
	}
	got := mapServiceUnitRef(&resolve.ResolvedServiceUnitRef{Name: "su1"})
	if got == nil || got.Name != "su1" {
		t.Fatalf("unexpected result: %+v", got)
	}
}
