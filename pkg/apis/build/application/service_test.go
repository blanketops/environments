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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	buildv1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/build/domain"
	bldResolution "github.com/blanketops/environments/resolution/build/resolve"
)

func newResolved(mutate func(*bldResolution.ResolvedBuildSpec)) *bldResolution.ResolvedBuild {
	spec := &bldResolution.ResolvedBuildSpec{
		Image:    "ghcr.io/acme/app:main",
		Source:   bldResolution.ResolvedSource{URL: "https://github.com/acme/app.git", Revision: "main", ContextDir: ".", CloneSecret: "git-creds"},
		Strategy: bldResolution.ResolvedStrategy{Name: "kaniko", StrategyKind: bldResolution.StrategyKindCluster},
		ServiceAccount: &bldResolution.ResolvedServiceAccount{
			Name: "build-bot", Secret: "registry-creds",
		},
	}
	if mutate != nil {
		mutate(spec)
	}
	return &bldResolution.ResolvedBuild{
		Build: &buildv1.Build{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"}},
		Spec:  spec,
	}
}

func TestMapper_MapsWhatTheContractDeclares(t *testing.T) {
	got, err := NewMapper().MapResolvedToDomain(newResolved(nil))
	if err != nil {
		t.Fatalf("MapResolvedToDomain: %v", err)
	}
	want := domain.BuildSpec{
		SourceURL: "https://github.com/acme/app.git", ContextDir: ".", Revision: "main", CloneSecret: "git-creds",
		StrategyName: "kaniko", StrategyKind: "ClusterBuildStrategy",
		Image:              "ghcr.io/acme/app:main",
		ServiceAccountName: "build-bot", ServiceAccountSecret: "registry-creds",
	}
	if got.SourceURL != want.SourceURL || got.ContextDir != want.ContextDir || got.Revision != want.Revision ||
		got.CloneSecret != want.CloneSecret || got.StrategyName != want.StrategyName || got.StrategyKind != want.StrategyKind ||
		got.Image != want.Image || got.ServiceAccountName != want.ServiceAccountName || got.ServiceAccountSecret != want.ServiceAccountSecret {
		t.Errorf("MapResolvedToDomain = %+v, want %+v", got, want)
	}
}

func TestMapper_StrategyKind(t *testing.T) {
	tests := []struct {
		declared string
		want     string
	}{
		{declared: bldResolution.StrategyKindCluster, want: "ClusterBuildStrategy"},
		{declared: bldResolution.StrategyKindNamespaced, want: "BuildStrategy"},
	}
	for _, tt := range tests {
		t.Run(tt.declared, func(t *testing.T) {
			got, err := NewMapper().MapResolvedToDomain(newResolved(func(s *bldResolution.ResolvedBuildSpec) {
				s.Strategy.StrategyKind = tt.declared
			}))
			if err != nil {
				t.Fatalf("MapResolvedToDomain: %v", err)
			}
			if got.StrategyKind != tt.want {
				t.Errorf("StrategyKind = %q, want %q", got.StrategyKind, tt.want)
			}
		})
	}
}

func TestMapper_OptionalServiceAccount(t *testing.T) {
	got, err := NewMapper().MapResolvedToDomain(newResolved(func(s *bldResolution.ResolvedBuildSpec) {
		s.ServiceAccount = nil
		s.Source.CloneSecret = ""
	}))
	if err != nil {
		t.Fatalf("MapResolvedToDomain: %v", err)
	}
	if got.ServiceAccountName != "" || got.ServiceAccountSecret != "" || got.CloneSecret != "" {
		t.Errorf("undeclared fields were filled in: %+v", got)
	}
}

// TestMapper_InvalidBuildIsAnError covers Builds that bypassed resolution.
// Each must come back as ErrInvalidBuild, not a panic.
func TestMapper_InvalidBuildIsAnError(t *testing.T) {
	tests := []struct {
		name     string
		resolved *bldResolution.ResolvedBuild
	}{
		{name: "nil resolved build", resolved: nil},
		{name: "no CR", resolved: &bldResolution.ResolvedBuild{Spec: &bldResolution.ResolvedBuildSpec{}}},
		{name: "no spec", resolved: &bldResolution.ResolvedBuild{Build: &buildv1.Build{}}},
		{name: "no source url", resolved: newResolved(func(s *bldResolution.ResolvedBuildSpec) { s.Source.URL = "" })},
		{name: "no strategy name", resolved: newResolved(func(s *bldResolution.ResolvedBuildSpec) { s.Strategy.Name = "" })},
		{name: "no strategy kind", resolved: newResolved(func(s *bldResolution.ResolvedBuildSpec) { s.Strategy.StrategyKind = "" })},
		{name: "unknown strategy kind", resolved: newResolved(func(s *bldResolution.ResolvedBuildSpec) { s.Strategy.StrategyKind = "BuildStrategy" })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMapper().MapResolvedToDomain(tt.resolved)
			if !errors.Is(err, domain.ErrInvalidBuild) {
				t.Fatalf("MapResolvedToDomain error = %v, want ErrInvalidBuild", err)
			}
		})
	}
}

// recordingProvider records which calls reached it.
type recordingProvider struct {
	runs, teardowns int
}

func (p *recordingProvider) Run(context.Context, *bldResolution.ResolvedBuild, domain.BuildSpec) (domain.BuildResult, error) {
	p.runs++
	return domain.BuildResult{Triggered: true}, nil
}

func (p *recordingProvider) Teardown(context.Context, *bldResolution.ResolvedBuild) error {
	p.teardowns++
	return nil
}

func TestBuildService_InvalidBuild(t *testing.T) {
	scheme := newStatusTestScheme(t)
	invalid := newResolved(func(s *bldResolution.ResolvedBuildSpec) { s.Strategy.Name = "" })
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(invalid.Build).WithStatusSubresource(invalid.Build).Build()

	buildah, kaniko, buildpacks := &recordingProvider{}, &recordingProvider{}, &recordingProvider{}
	svc := NewBuildService(NewMapper(), NewStatusWriter(c, logr.Discard()), NewBackendSelector(buildah, kaniko, buildpacks))

	// Reconcile refuses to dispatch a Build it cannot map.
	if err := svc.Reconcile(context.Background(), invalid); !errors.Is(err, domain.ErrInvalidBuild) {
		t.Fatalf("Reconcile error = %v, want ErrInvalidBuild", err)
	}
	if buildah.runs+kaniko.runs+buildpacks.runs != 0 {
		t.Fatal("an invalid Build was dispatched to a provider")
	}

	// Teardown still runs, so the Build can be deleted.
	if err := svc.Teardown(context.Background(), invalid); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if buildah.teardowns != 1 {
		t.Fatalf("Teardown reached the default provider %d times, want 1", buildah.teardowns)
	}
}

func TestBuildService_ValidBuildIsDispatched(t *testing.T) {
	scheme := newStatusTestScheme(t)
	valid := newResolved(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(valid.Build).WithStatusSubresource(valid.Build).Build()

	buildah, kaniko, buildpacks := &recordingProvider{}, &recordingProvider{}, &recordingProvider{}
	svc := NewBuildService(NewMapper(), NewStatusWriter(c, logr.Discard()), NewBackendSelector(buildah, kaniko, buildpacks))

	if err := svc.Reconcile(context.Background(), valid); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if kaniko.runs != 1 || buildah.runs != 0 || buildpacks.runs != 0 {
		t.Fatalf("runs: kaniko=%d buildah=%d buildpacks=%d, want only kaniko", kaniko.runs, buildah.runs, buildpacks.runs)
	}
	if err := svc.Teardown(context.Background(), valid); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if kaniko.teardowns != 1 {
		t.Fatalf("kaniko teardowns = %d, want 1", kaniko.teardowns)
	}
}
