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

	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	r := &sourcesv1alpha1.GitRepository{
		Spec: sourcesv1alpha1.GitRepositorySpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"provider":"github","hookUrl":"https://hooks/x","repository":{"owner":"blanketops","name":"environments"}}`)},
		},
	}
	resolved, err := a.Resolve(context.Background(), r)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Provider != "github" {
		t.Fatalf("unexpected Provider: %q", resolved.Spec.Provider)
	}
}

func TestAdapter_Resolve_PropagatesError(t *testing.T) {
	a := NewAdapter()
	if _, err := a.Resolve(context.Background(), &sourcesv1alpha1.GitRepository{}); err == nil {
		t.Fatal("expected error to propagate from resolve.ResolveGitRepository")
	}
}
