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

package adapter

import (
	"context"
	"testing"

	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	d := &networksv1alpha1.Domain{
		Spec: networksv1alpha1.DomainSpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"host":"app.dev.blanketops.online","routeRef":{"name":"route1"},"tlsStrategy":"platform"}`)},
		},
	}
	resolved, err := a.Resolve(context.Background(), d)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Host != "app.dev.blanketops.online" {
		t.Fatalf("unexpected Host: %q", resolved.Spec.Host)
	}
}

func TestAdapter_Resolve_PropagatesError(t *testing.T) {
	a := NewAdapter()
	if _, err := a.Resolve(context.Background(), &networksv1alpha1.Domain{}); err == nil {
		t.Fatal("expected error to propagate from resolve.ResolveDomain")
	}
}
