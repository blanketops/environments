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

package route

import (
	"context"
	"strings"
	"testing"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func routeWithContract(raw string) *networksv1alpha1.Route {
	return &networksv1alpha1.Route{
		Spec: networksv1alpha1.RouteSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"host":"app.dev.blanketops.online","runtime":"kubernetes-container","serviceUnitRef":{"name":"su1"}}`

func TestResolveRoute_Nil(t *testing.T) {
	if _, err := ResolveRoute(nil); err == nil {
		t.Fatal("expected error for nil route")
	}
}

func TestResolveRoute_EmptyContract(t *testing.T) {
	r := &networksv1alpha1.Route{}
	if _, err := ResolveRoute(r); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveRoute_InvalidJSON(t *testing.T) {
	r := routeWithContract(`{not json`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveRoute_HostMissing(t *testing.T) {
	r := routeWithContract(`{"runtime":"kubernetes-container","serviceUnitRef":{"name":"su1"}}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), `"host"`) {
		t.Fatalf("expected host error, got %v", err)
	}
}

func TestResolveRoute_RuntimeMissing(t *testing.T) {
	r := routeWithContract(`{"host":"x","serviceUnitRef":{"name":"su1"}}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), `"runtime"`) {
		t.Fatalf("expected runtime error, got %v", err)
	}
}

func TestResolveRoute_ServiceUnitRefMissing(t *testing.T) {
	r := routeWithContract(`{"host":"x","runtime":"kubernetes-container"}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), `"serviceUnitRef"`) {
		t.Fatalf("expected serviceUnitRef error, got %v", err)
	}
}

func TestResolveRoute_ServiceUnitRefWrongType(t *testing.T) {
	r := routeWithContract(`{"host":"x","runtime":"kubernetes-container","serviceUnitRef":"not-an-object"}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), `"serviceUnitRef" must be an object`) {
		t.Fatalf("expected serviceUnitRef type error, got %v", err)
	}
}

func TestResolveRoute_ServiceUnitRefMissingName(t *testing.T) {
	r := routeWithContract(`{"host":"x","runtime":"kubernetes-container","serviceUnitRef":{}}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), `"serviceUnitRef.name" must be a non-empty string`) {
		t.Fatalf("expected serviceUnitRef.name error, got %v", err)
	}
}

func TestResolveRoute_MinimalValid(t *testing.T) {
	r := routeWithContract(minimalValid)
	resolved, err := ResolveRoute(r)
	if err != nil {
		t.Fatalf("ResolveRoute: %v", err)
	}
	if resolved.Spec.Host != "app.dev.blanketops.online" {
		t.Fatalf("unexpected Host: %q", resolved.Spec.Host)
	}
	if !resolved.Spec.Enabled {
		t.Fatal("expected Enabled to default to true")
	}
	if resolved.Spec.Path != "/" {
		t.Fatalf("expected Path to default to \"/\", got %q", resolved.Spec.Path)
	}
	if !resolved.Spec.TLSEnabled {
		t.Fatal("expected TLSEnabled to default to true")
	}
	if resolved.Spec.Runtime == nil || *resolved.Spec.Runtime != RuntimeKubernetesContainer {
		t.Fatalf("unexpected Runtime: %v", resolved.Spec.Runtime)
	}
	if resolved.Spec.ServiceUnitRef.Name != "su1" {
		t.Fatalf("unexpected ServiceUnitRef: %+v", resolved.Spec.ServiceUnitRef)
	}
}

func TestResolveRoute_ExplicitOverrides(t *testing.T) {
	r := routeWithContract(`{"host":"x","runtime":"knative-service","serviceUnitRef":{"name":"su1"},"enabled":false,"path":"/api","tlsEnabled":false}`)
	resolved, err := ResolveRoute(r)
	if err != nil {
		t.Fatalf("ResolveRoute: %v", err)
	}
	if resolved.Spec.Enabled {
		t.Fatal("expected Enabled to be false")
	}
	if resolved.Spec.Path != "/api" {
		t.Fatalf("unexpected Path: %q", resolved.Spec.Path)
	}
	if resolved.Spec.TLSEnabled {
		t.Fatal("expected TLSEnabled to be false")
	}
	if *resolved.Spec.Runtime != RuntimeKnativeService {
		t.Fatalf("unexpected Runtime: %v", *resolved.Spec.Runtime)
	}
}

func TestResolveRoute_HostWrongType(t *testing.T) {
	r := routeWithContract(`{"host":5,"runtime":"kubernetes-container","serviceUnitRef":{"name":"su1"}}`)
	_, err := ResolveRoute(r)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	r := routeWithContract(minimalValid)
	resolved, err := a.Resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Host != "app.dev.blanketops.online" {
		t.Fatalf("unexpected Host: %q", resolved.Spec.Host)
	}
}
