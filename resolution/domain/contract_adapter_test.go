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

package domain

import (
	"testing"

	commonv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

func TestToDomainContract_NilDomain(t *testing.T) {
	out, err := ToDomainContract(nil)
	if err != nil || out != nil {
		t.Fatalf("expected (nil, nil), got (%+v, %v)", out, err)
	}
}

func TestToDomainContract_NilSpec(t *testing.T) {
	out, err := ToDomainContract(&ResolvedDomain{})
	if err != nil || out != nil {
		t.Fatalf("expected (nil, nil), got (%+v, %v)", out, err)
	}
}

func TestToDomainContract_Populated(t *testing.T) {
	ts := TLSStrategyCustom
	d := &ResolvedDomain{
		Spec: &ResolvedDomainSpec{
			Host:        "x",
			RouteRef:    ResolvedDomainRouteRef{Name: "route1"},
			TLSStrategy: &ts,
			MTLS:        ResolvedDomainMTLS{Enforced: true},
			RenewBefore: "720h",
		},
	}
	out, err := ToDomainContract(d)
	if err != nil {
		t.Fatalf("ToDomainContract: %v", err)
	}
	if out.Host != "x" || out.RouteRef.Name != "route1" || out.RenewBefore != "720h" {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.TlsStrategy.Strategy != commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_CUSTOM {
		t.Fatalf("unexpected TlsStrategy: %v", out.TlsStrategy.Strategy)
	}
	if !out.Mtls.Enforced {
		t.Fatal("expected Mtls.Enforced to be true")
	}
}

func TestMapTLSStrategy(t *testing.T) {
	platform := TLSStrategyPlatform
	custom := TLSStrategyCustom
	unknown := TLSStrategy("bogus")

	cases := []struct {
		name string
		ts   *TLSStrategy
		want commonv1.DomainTLSStrategy_DomainTLSStrategy
	}{
		{"nil", nil, commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_UNSPECIFIED},
		{"platform", &platform, commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_PLATFORM},
		{"custom", &custom, commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_CUSTOM},
		{"unknown", &unknown, commonv1.DomainTLSStrategy_DOMAIN_TLS_STRATEGY_UNSPECIFIED},
	}
	for _, tc := range cases {
		if got := mapTLSStrategy(tc.ts); got.Strategy != tc.want {
			t.Fatalf("%s: mapTLSStrategy = %v, want %v", tc.name, got.Strategy, tc.want)
		}
	}
}

func TestMapRouteRef(t *testing.T) {
	got := mapRouteRef(ResolvedDomainRouteRef{Name: "route1"})
	if got.Name != "route1" {
		t.Fatalf("unexpected RouteRef: %+v", got)
	}
}

func TestMapMTLS(t *testing.T) {
	got := mapMTLS(ResolvedDomainMTLS{Enforced: true})
	if !got.Enforced {
		t.Fatal("expected Enforced to be true")
	}
}
