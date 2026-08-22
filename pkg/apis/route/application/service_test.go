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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/route/domain"
	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

type fakeProvider struct {
	ensureResult   domain.RouteResult
	ensureErr      error
	teardownCalled bool
	teardownRoute  domain.Route
	teardownErr    error
}

func (f *fakeProvider) Ensure(ctx context.Context, resolved *routeResolution.ResolvedRoute, route domain.Route) (domain.RouteResult, error) {
	return f.ensureResult, f.ensureErr
}

func (f *fakeProvider) Teardown(ctx context.Context, route domain.Route) error {
	f.teardownCalled = true
	f.teardownRoute = route
	return f.teardownErr
}

func testResolvedRoute(t *testing.T, runtime routeResolution.Runtime) *routeResolution.ResolvedRoute {
	t.Helper()
	return &routeResolution.ResolvedRoute{
		Route: &networksv1alpha1.Route{
			ObjectMeta: metav1.ObjectMeta{Name: "my-route", Namespace: "default"},
		},
		Spec: &routeResolution.ResolvedRouteSpec{
			Host:           "app.dev.blanketops.online",
			Enabled:        true,
			Path:           "/",
			TLSEnabled:     true,
			Runtime:        &runtime,
			ServiceUnitRef: &routeResolution.ResolvedServiceUnitRef{Name: "api"},
		},
	}
}

func TestRouteService_Teardown_DispatchesToSelectedProvider(t *testing.T) {
	knative := &fakeProvider{}
	ingress := &fakeProvider{}
	selector := NewBackendSelector(knative, ingress)
	svc := NewRouteService(NewMapper(), nil, selector)

	resolved := testResolvedRoute(t, routeResolution.RuntimeKnativeService)

	if err := svc.Teardown(context.Background(), resolved); err != nil {
		t.Fatalf("Teardown: unexpected error: %v", err)
	}

	if !knative.teardownCalled {
		t.Error("expected Teardown to dispatch to the Knative provider for RuntimeKnativeService")
	}
	if ingress.teardownCalled {
		t.Error("expected Teardown NOT to dispatch to the Ingress provider")
	}
	if knative.teardownRoute.Name != "my-route" {
		t.Errorf("teardownRoute.Name = %q, want %q", knative.teardownRoute.Name, "my-route")
	}
}

func TestRouteService_Teardown_UnknownRuntimeReturnsSelectorError(t *testing.T) {
	knative := &fakeProvider{}
	ingress := &fakeProvider{}
	selector := NewBackendSelector(knative, ingress)
	svc := NewRouteService(NewMapper(), nil, selector)

	resolved := testResolvedRoute(t, routeResolution.Runtime("mystery-runtime"))

	err := svc.Teardown(context.Background(), resolved)
	if err == nil {
		t.Fatal("expected an error for an unrecognised runtime, got nil")
	}
	if knative.teardownCalled || ingress.teardownCalled {
		t.Error("expected no provider's Teardown to be called when backend selection fails")
	}
}

func TestRouteService_Teardown_PropagatesProviderError(t *testing.T) {
	wantErr := context.DeadlineExceeded
	knative := &fakeProvider{teardownErr: wantErr}
	ingress := &fakeProvider{}
	selector := NewBackendSelector(knative, ingress)
	svc := NewRouteService(NewMapper(), nil, selector)

	resolved := testResolvedRoute(t, routeResolution.RuntimeKnativeService)

	err := svc.Teardown(context.Background(), resolved)
	if err != wantErr {
		t.Fatalf("Teardown error = %v, want %v", err, wantErr)
	}
}
