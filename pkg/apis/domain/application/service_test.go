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
	"github.com/blanketops/environments/pkg/apis/domain/domain"
	domainResolution "github.com/blanketops/environments/resolution/domain/resolve"
)

type fakeProvider struct {
	ensureResult   domain.DomainResult
	ensureErr      error
	teardownCalled bool
	teardownDomain domain.Domain
	teardownErr    error
}

func (f *fakeProvider) Ensure(ctx context.Context, resolved *domainResolution.ResolvedDomain, d domain.Domain) (domain.DomainResult, error) {
	return f.ensureResult, f.ensureErr
}

func (f *fakeProvider) Teardown(ctx context.Context, d domain.Domain) error {
	f.teardownCalled = true
	f.teardownDomain = d
	return f.teardownErr
}

func testResolvedDomain(t *testing.T, strategy domainResolution.TLSStrategy) *domainResolution.ResolvedDomain {
	t.Helper()
	return &domainResolution.ResolvedDomain{
		Domain: &networksv1alpha1.Domain{
			ObjectMeta: metav1.ObjectMeta{Name: "my-domain", Namespace: "default"},
		},
		Spec: &domainResolution.ResolvedDomainSpec{
			Host:        "app.dev.blanketops.online",
			RouteRef:    domainResolution.ResolvedDomainRouteRef{Name: "my-route"},
			TLSStrategy: &strategy,
		},
	}
}

func TestDomainService_Teardown_DispatchesToKnativeProvider(t *testing.T) {
	knative := &fakeProvider{}
	selector := NewBackendSelector(knative)
	svc := NewDomainService(NewMapper(), nil, selector)

	resolved := testResolvedDomain(t, domainResolution.TLSStrategyPlatform)

	if err := svc.Teardown(context.Background(), resolved); err != nil {
		t.Fatalf("Teardown: unexpected error: %v", err)
	}

	if !knative.teardownCalled {
		t.Error("expected Teardown to dispatch to the Knative provider")
	}
	if knative.teardownDomain.Name != "my-domain" {
		t.Errorf("teardownDomain.Name = %q, want %q", knative.teardownDomain.Name, "my-domain")
	}
}

func TestDomainService_Teardown_UnknownStrategyReturnsSelectorError(t *testing.T) {
	knative := &fakeProvider{}
	selector := NewBackendSelector(knative)
	svc := NewDomainService(NewMapper(), nil, selector)

	resolved := testResolvedDomain(t, domainResolution.TLSStrategy("mystery-strategy"))

	err := svc.Teardown(context.Background(), resolved)
	if err == nil {
		t.Fatal("expected an error for an unrecognised strategy, got nil")
	}
	if knative.teardownCalled {
		t.Error("expected the provider's Teardown NOT to be called when backend selection fails")
	}
}

func TestDomainService_Teardown_PropagatesProviderError(t *testing.T) {
	wantErr := context.DeadlineExceeded
	knative := &fakeProvider{teardownErr: wantErr}
	selector := NewBackendSelector(knative)
	svc := NewDomainService(NewMapper(), nil, selector)

	resolved := testResolvedDomain(t, domainResolution.TLSStrategyCustom)

	err := svc.Teardown(context.Background(), resolved)
	if err != wantErr {
		t.Fatalf("Teardown error = %v, want %v", err, wantErr)
	}
}
