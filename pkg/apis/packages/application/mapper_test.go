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
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"
	pkgintent "github.com/blanketops/environments/pkg/intent/package"
	pkgResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

func TestMapResolvedToDomain_NilStateRepository_DefaultsToPlainYAML(t *testing.T) {
	rp := &pkgResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: "my-package", Namespace: "default"},
		},
		Spec: &pkgResolution.ResolvedPackageSpec{
			Name:    "my-package",
			Version: "1.0.0",
			// StateRepository intentionally left nil — the optional-field case.
		},
	}

	got, err := Mapper{}.MapResolvedToDomain(rp)
	if err != nil {
		t.Fatalf("MapResolvedToDomain: %v", err)
	}

	if got.Strategy != domain.StrategyPlainYAML {
		t.Fatalf("expected StrategyPlainYAML for a Package with no stateRepository, got %v", got.Strategy)
	}
	if got.StateRepo != (domain.StateRepository{}) {
		t.Fatalf("expected a zero-value StateRepo, got %+v", got.StateRepo)
	}
}

// teardownRecorder is a Provider that records the identities it was asked to
// tear down.
type teardownRecorder struct {
	tornDown []domain.PackageID
}

func (p *teardownRecorder) Execute(context.Context, *pkgintent.PackageIntent) (*domain.PackageResult, error) {
	return &domain.PackageResult{}, nil
}

func (p *teardownRecorder) Teardown(_ context.Context, id domain.PackageID) error {
	p.tornDown = append(p.tornDown, id)
	return nil
}

func TestPackageService_TeardownReachesTheProvider(t *testing.T) {
	provider := &teardownRecorder{}
	svc := NewPackageService(NewMapper(), NewBackendSelector(provider), nil)

	id := domain.PackageID{Namespace: "default", Name: "app-package"}
	if err := svc.Teardown(context.Background(), id); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if len(provider.tornDown) != 1 || provider.tornDown[0] != id {
		t.Errorf("provider was asked to tear down %v, want [%v]", provider.tornDown, id)
	}
}

func resolvedForMapper(mutate func(*pkgResolution.ResolvedPackageSpec)) *pkgResolution.ResolvedPackage {
	spec := &pkgResolution.ResolvedPackageSpec{
		Name: "app", Version: "1.0.0",
		PackageRepository: pkgResolution.ResolvedPackageRepository{URL: "git@github.com:example-org/packages.git", CredentialsSecret: "creds"},
		StateRepository: &pkgResolution.ResolvedStateRepository{
			URL: "git@github.com:example-org/state.git", Ref: "master", CloneSecret: "state-creds",
			Strategy: "kustomization", Path: "./clusters/dev",
		},
		Maintainers: []pkgResolution.ResolvedMaintainer{{Name: "Neo", Email: "neo@example.com"}},
		DiffEnabled: true,
	}
	if mutate != nil {
		mutate(spec)
	}
	return &pkgResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{ObjectMeta: metav1.ObjectMeta{Name: "app-package", Namespace: "default"}},
		Spec:    spec,
	}
}

func TestMapResolvedToDomain_MapsWhatTheContractDeclares(t *testing.T) {
	got, err := NewMapper().MapResolvedToDomain(resolvedForMapper(nil))
	if err != nil {
		t.Fatalf("MapResolvedToDomain: %v", err)
	}
	if got.ID != (domain.PackageID{Namespace: "default", Name: "app-package"}) || got.Name != "app" || got.Version != "1.0.0" || !got.DiffEnabled {
		t.Errorf("unexpected identity or flags: %+v", got)
	}
	if got.Source.RepositoryURL != "git@github.com:example-org/packages.git" || got.Source.CredentialsSecret != "creds" {
		t.Errorf("source = %+v", got.Source)
	}
	if got.StateRepo.Ref != "master" || got.StateRepo.Path != "./clusters/dev" || got.Strategy != domain.StrategyKustomize {
		t.Errorf("state repo = %+v strategy = %q", got.StateRepo, got.Strategy)
	}
	if len(got.Maintainers) != 1 || got.Maintainers[0].Email != "neo@example.com" {
		t.Errorf("maintainers = %+v", got.Maintainers)
	}
}

// TestMapResolvedToDomain_InvalidIsAnError covers resolved Packages that
// bypassed resolution. Each must be an InvalidSpecError, never a panic.
func TestMapResolvedToDomain_InvalidIsAnError(t *testing.T) {
	tests := []struct {
		name     string
		resolved *pkgResolution.ResolvedPackage
	}{
		{name: "nil", resolved: nil},
		{name: "no CR", resolved: &pkgResolution.ResolvedPackage{Spec: &pkgResolution.ResolvedPackageSpec{}}},
		{name: "no spec", resolved: &pkgResolution.ResolvedPackage{Package: &environmentv1alpha1.Package{}}},
		{name: "no name", resolved: resolvedForMapper(func(s *pkgResolution.ResolvedPackageSpec) { s.Name = "" })},
		{name: "no version", resolved: resolvedForMapper(func(s *pkgResolution.ResolvedPackageSpec) { s.Version = "" })},
		{name: "unknown strategy", resolved: resolvedForMapper(func(s *pkgResolution.ResolvedPackageSpec) { s.StateRepository.Strategy = "helm" })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMapper().MapResolvedToDomain(tt.resolved)
			var invalid domain.InvalidSpecError
			if !errors.As(err, &invalid) {
				t.Fatalf("MapResolvedToDomain error = %v, want an InvalidSpecError", err)
			}
		})
	}
}

// scriptedProvider returns a fixed result and records whether it ran.
type scriptedProvider struct {
	result   *domain.PackageResult
	err      error
	executed int
}

func (p *scriptedProvider) Execute(context.Context, *pkgintent.PackageIntent) (*domain.PackageResult, error) {
	p.executed++
	return p.result, p.err
}

func (p *scriptedProvider) Teardown(context.Context, domain.PackageID) error { return nil }

func newStatusScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return scheme
}

func readPackageStatus(t *testing.T, c client.Client) (domain.PackageStatus, []metav1.Condition) {
	t.Helper()
	var p environmentv1alpha1.Package
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "app-package"}, &p); err != nil {
		t.Fatalf("get package: %v", err)
	}
	var st domain.PackageStatus
	if len(p.Status.Contract.Raw) > 0 {
		if err := json.Unmarshal(p.Status.Contract.Raw, &st); err != nil {
			t.Fatalf("decode contract: %v", err)
		}
	}
	return st, p.Status.Conditions
}

// TestPackageService_ReconcileOutcomes covers what each execution outcome
// leaves on the Package. A package that is still being applied is not
// reported as failed.
func TestPackageService_ReconcileOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		result     *domain.PackageResult
		runErr     error
		wantStatus metav1.ConditionStatus
		wantReason string
		wantPhase  domain.PackagePhase
		wantOK     bool
	}{
		{
			name:       "applied",
			result:     &domain.PackageResult{Success: true, Phase: domain.PackagePhaseSucceeded},
			wantStatus: metav1.ConditionTrue, wantReason: "PackageApplied", wantPhase: domain.PackagePhaseSucceeded, wantOK: true,
		},
		{
			name:       "still being applied",
			result:     &domain.PackageResult{Phase: domain.PackagePhasePending},
			wantStatus: metav1.ConditionUnknown, wantReason: "PackagePending", wantPhase: domain.PackagePhasePending,
		},
		{
			name:       "app reports failure",
			result:     &domain.PackageResult{Phase: domain.PackagePhaseFailed, Message: "fetch failed"},
			wantStatus: metav1.ConditionFalse, wantReason: "PackageFailed", wantPhase: domain.PackagePhaseFailed,
		},
		{
			name:       "provider error",
			result:     &domain.PackageResult{Phase: domain.PackagePhasePending},
			runErr:     errors.New("apply rejected"),
			wantStatus: metav1.ConditionFalse, wantReason: "PackageFailed", wantPhase: domain.PackagePhaseFailed,
		},
		{
			name:       "provider error without a result",
			runErr:     errors.New("apply rejected"),
			wantStatus: metav1.ConditionFalse, wantReason: "PackageFailed", wantPhase: domain.PackagePhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := resolvedForMapper(nil)
			// An earlier condition from another writer must survive.
			resolved.Package.Status.Conditions = []metav1.Condition{{
				Type: "PackageResolved", Status: metav1.ConditionTrue, Reason: "Resolved", Message: "resolved",
			}}
			scheme := newStatusScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(resolved.Package).WithStatusSubresource(resolved.Package).Build()
			provider := &scriptedProvider{result: tt.result, err: tt.runErr}
			svc := NewPackageService(NewMapper(), NewBackendSelector(provider), NewStatusWriter(c, logr.Discard()))

			// The caller's copy is stale: it has no conditions.
			stale := resolvedForMapper(nil)
			if err := svc.Reconcile(context.Background(), stale, &pkgintent.PackageIntent{}); err != nil {
				t.Fatalf("Reconcile: %v", err)
			}
			if provider.executed != 1 {
				t.Fatalf("provider ran %d times, want 1", provider.executed)
			}

			st, conds := readPackageStatus(t, c)
			if st.Phase != tt.wantPhase || st.Success != tt.wantOK {
				t.Errorf("contract = %+v, want phase %s success %v", st, tt.wantPhase, tt.wantOK)
			}
			succeeded := apimeta.FindStatusCondition(conds, "Succeeded")
			if succeeded == nil || succeeded.Status != tt.wantStatus || succeeded.Reason != tt.wantReason {
				t.Errorf("Succeeded = %+v, want %s/%s", succeeded, tt.wantStatus, tt.wantReason)
			}
			if !apimeta.IsStatusConditionTrue(conds, "PackageResolved") {
				t.Errorf("a condition written earlier was lost: %+v", conds)
			}
		})
	}
}

func TestPackageService_RefusesWhatItCannotRun(t *testing.T) {
	resolved := resolvedForMapper(nil)
	intent := &pkgintent.PackageIntent{}

	noProvider := NewPackageService(NewMapper(), NewBackendSelector(nil), nil)
	if err := noProvider.Reconcile(context.Background(), resolved, intent); err == nil {
		t.Error("Reconcile without a provider: want an error")
	}
	if err := noProvider.Teardown(context.Background(), domain.PackageID{Name: "x"}); err == nil {
		t.Error("Teardown without a provider: want an error")
	}

	svc := NewPackageService(NewMapper(), NewBackendSelector(&scriptedProvider{}), nil)
	for name, call := range map[string]func() error{
		"nil resolved": func() error { return svc.Reconcile(context.Background(), nil, intent) },
		"nil intent":   func() error { return svc.Reconcile(context.Background(), resolved, nil) },
	} {
		if err := call(); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
