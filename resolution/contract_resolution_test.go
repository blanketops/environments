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

package resolution

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
)

func TestAdapter_Resolve_Build(t *testing.T) {
	a := NewAdapter()
	b := &environmentv1alpha1.Build{
		Spec: environmentv1alpha1.BuildSpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"image":"foo","source":{"url":"x"}}`)},
		},
	}
	if err := a.Resolve(context.Background(), b); err != nil {
		t.Fatalf("Resolve(Build): %v", err)
	}
}

func TestAdapter_Resolve_Build_PropagatesResolutionError(t *testing.T) {
	a := NewAdapter()
	b := &environmentv1alpha1.Build{} // empty contract -> resolution error
	if err := a.Resolve(context.Background(), b); err == nil {
		t.Fatal("expected error to propagate from ResolveBuild")
	}
}

func TestAdapter_Resolve_Deployment(t *testing.T) {
	a := NewAdapter()
	d := &environmentv1alpha1.Deployment{
		Spec: environmentv1alpha1.DeploymentSpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling"}`)},
		},
	}
	if err := a.Resolve(context.Background(), d); err != nil {
		t.Fatalf("Resolve(Deployment): %v", err)
	}
}

func TestAdapter_Resolve_GitHubEvent(t *testing.T) {
	a := NewAdapter()
	ev := &eventsv1alpha1.GitHubEvent{
		Spec: eventsv1alpha1.GitHubEventSpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"repository":"x/y","eventType":"push"}`)},
		},
	}
	if err := a.Resolve(context.Background(), ev); err != nil {
		t.Fatalf("Resolve(GitHubEvent): %v", err)
	}
}

func TestAdapter_Resolve_GitRepository(t *testing.T) {
	a := NewAdapter()
	r := &sourcesv1alpha1.GitRepository{
		Spec: sourcesv1alpha1.GitRepositorySpec{
			Contract: runtime.RawExtension{Raw: []byte(`{"provider":"github","hookUrl":"x","repository":{"owner":"o","name":"n"}}`)},
		},
	}
	if err := a.Resolve(context.Background(), r); err != nil {
		t.Fatalf("Resolve(GitRepository): %v", err)
	}
}

// Package now reaches the wired-up packages.Adapter instead of falling
// through to the "unsupported object type" default. A zero-value Package
// fails resolution's own spec validation ("spec.contract is required"),
// proving dispatch reached ResolvePackage rather than being rejected at
// the type switch.
func TestAdapter_Resolve_Package_DispatchesToPackagesAdapter(t *testing.T) {
	a := NewAdapter()
	p := &environmentv1alpha1.Package{}
	err := a.Resolve(context.Background(), p)
	if err == nil {
		t.Fatal("expected an error from resolution's own validation")
	}
	if strings.Contains(err.Error(), "unsupported object type") {
		t.Fatalf("Package should no longer fall through to the unsupported-object-type default, got %v", err)
	}
	if !strings.Contains(err.Error(), "spec.contract") {
		t.Fatalf("expected a spec.contract validation error from ResolvePackage, got %v", err)
	}
}

func TestAdapter_Resolve_UnsupportedType(t *testing.T) {
	a := NewAdapter()
	if err := a.Resolve(context.Background(), &corev1.Pod{}); err == nil {
		t.Fatal("expected error for unsupported object type")
	}
}
