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
	"strings"
	"testing"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func domainWithContract(raw string) *networksv1alpha1.Domain {
	return &networksv1alpha1.Domain{
		Spec: networksv1alpha1.DomainSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"host":"app.dev.blanketops.online","routeRef":{"name":"route1"},"tlsStrategy":"platform"}`

func TestResolveDomain_Nil(t *testing.T) {
	if _, err := ResolveDomain(nil); err == nil {
		t.Fatal("expected error for nil domain")
	}
}

func TestResolveDomain_EmptyContract(t *testing.T) {
	d := &networksv1alpha1.Domain{}
	if _, err := ResolveDomain(d); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveDomain_InvalidJSON(t *testing.T) {
	d := domainWithContract(`{not json`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveDomain_HostMissing(t *testing.T) {
	d := domainWithContract(`{"routeRef":{"name":"route1"},"tlsStrategy":"platform"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `"host"`) {
		t.Fatalf("expected host error, got %v", err)
	}
}

func TestResolveDomain_RouteRefMissing(t *testing.T) {
	d := domainWithContract(`{"host":"x","tlsStrategy":"platform"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `"routeRef"`) {
		t.Fatalf("expected routeRef error, got %v", err)
	}
}

func TestResolveDomain_RouteRefWrongType(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":"not-an-object","tlsStrategy":"platform"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `"routeRef" must be an object`) {
		t.Fatalf("expected routeRef type error, got %v", err)
	}
}

func TestResolveDomain_RouteRefMissingName(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{},"tlsStrategy":"platform"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), "routeRef.name") {
		t.Fatalf("expected routeRef.name error, got %v", err)
	}
}

func TestResolveDomain_TLSStrategyMissing(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{"name":"route1"}}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `"tlsStrategy"`) {
		t.Fatalf("expected tlsStrategy error, got %v", err)
	}
}

func TestResolveDomain_TLSStrategyInvalid(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{"name":"route1"},"tlsStrategy":"bogus"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `must be "platform" or "custom"`) {
		t.Fatalf("expected tlsStrategy validation error, got %v", err)
	}
}

func TestResolveDomain_MinimalValid(t *testing.T) {
	d := domainWithContract(minimalValid)
	resolved, err := ResolveDomain(d)
	if err != nil {
		t.Fatalf("ResolveDomain: %v", err)
	}
	if resolved.Spec.Host != "app.dev.blanketops.online" {
		t.Fatalf("unexpected Host: %q", resolved.Spec.Host)
	}
	if resolved.Spec.RouteRef.Name != "route1" {
		t.Fatalf("unexpected RouteRef: %+v", resolved.Spec.RouteRef)
	}
	if resolved.Spec.TLSStrategy == nil || *resolved.Spec.TLSStrategy != TLSStrategyPlatform {
		t.Fatalf("unexpected TLSStrategy: %v", resolved.Spec.TLSStrategy)
	}
	if resolved.Spec.MTLS.Enforced {
		t.Fatal("expected MTLS.Enforced to default to false")
	}
	if resolved.Spec.RenewBefore != "" {
		t.Fatalf("expected empty RenewBefore, got %q", resolved.Spec.RenewBefore)
	}
}

func TestResolveDomain_TLSStrategyCustom(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{"name":"route1"},"tlsStrategy":"custom","renewBefore":"720h"}`)
	resolved, err := ResolveDomain(d)
	if err != nil {
		t.Fatalf("ResolveDomain: %v", err)
	}
	if *resolved.Spec.TLSStrategy != TLSStrategyCustom {
		t.Fatalf("unexpected TLSStrategy: %v", *resolved.Spec.TLSStrategy)
	}
	if resolved.Spec.RenewBefore != "720h" {
		t.Fatalf("unexpected RenewBefore: %q", resolved.Spec.RenewBefore)
	}
}

func TestResolveDomain_MTLSWrongType(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{"name":"route1"},"tlsStrategy":"platform","mtls":"not-an-object"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), `"mtls" must be an object`) {
		t.Fatalf("expected mtls type error, got %v", err)
	}
}

func TestResolveDomain_MTLSEnforced(t *testing.T) {
	d := domainWithContract(`{"host":"x","routeRef":{"name":"route1"},"tlsStrategy":"platform","mtls":{"enforced":true}}`)
	resolved, err := ResolveDomain(d)
	if err != nil {
		t.Fatalf("ResolveDomain: %v", err)
	}
	if !resolved.Spec.MTLS.Enforced {
		t.Fatal("expected MTLS.Enforced to be true")
	}
}

func TestResolveDomain_HostWrongType(t *testing.T) {
	d := domainWithContract(`{"host":5,"routeRef":{"name":"route1"},"tlsStrategy":"platform"}`)
	_, err := ResolveDomain(d)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}
