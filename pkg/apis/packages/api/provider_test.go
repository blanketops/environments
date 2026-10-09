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

package api

import (
	"context"
	"testing"

	kappctrlv1alpha1 "carvel.dev/kapp-controller/pkg/apis/kappctrl/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/blanketops/environments/pkg/apis/packages/domain"
	intent "github.com/blanketops/environments/pkg/intent/package"
)

func newPackageScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := kappctrlv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(kappctrl): %v", err)
	}
	return scheme
}

// packageProviders lists every Provider so lifecycle behaviour is asserted
// identically across them.
func packageProviders(c client.Client, scheme *runtime.Scheme) map[string]Provider {
	log := logr.Discard()
	return map[string]Provider{
		"application": NewApplicationProvider(c, scheme, log, nil),
		"package":     NewPackageProvider(c, scheme, log, nil),
	}
}

func newPackageIntent() *intent.PackageIntent {
	return &intent.PackageIntent{
		ID:       domain.PackageID{Namespace: "default", Name: "app-package"},
		OwnerUID: types.UID("uid-package"),
		Labels: map[string]string{
			"environments.blanketops.dev/name": "app",
			"environments.blanketops.dev/type": "dev",
		},
		Source: domain.PackageSource{RepositoryURL: "git@github.com:example-org/packages.git"},
	}
}

func getApp(t *testing.T, c client.Client, id domain.PackageID) (*kappctrlv1alpha1.App, error) {
	t.Helper()
	app := &kappctrlv1alpha1.App{}
	err := c.Get(context.Background(), client.ObjectKey{Namespace: id.Namespace, Name: id.Name}, app)
	return app, err
}

func TestExecute_AppIsOwnedAndLabelled(t *testing.T) {
	scheme := newPackageScheme(t)
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := packageProviders(c, scheme)[name]
			in := newPackageIntent()

			if _, err := p.Execute(context.Background(), in); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			app, err := getApp(t, c, in.ID)
			if err != nil {
				t.Fatalf("get app: %v", err)
			}
			if app.Labels["environments.blanketops.dev/name"] != "app" || app.Labels["environments.blanketops.dev/type"] != "dev" {
				t.Errorf("labels = %v, want the Package's environments.blanketops.dev labels", app.Labels)
			}
			if len(app.OwnerReferences) != 1 {
				t.Fatalf("owner references = %+v, want one", app.OwnerReferences)
			}
			owner := app.OwnerReferences[0]
			if owner.Kind != "Package" || owner.Name != in.ID.Name || owner.UID != in.OwnerUID ||
				owner.Controller == nil || !*owner.Controller {
				t.Errorf("owner reference = %+v, want the controlling Package", owner)
			}
			if app.Spec.Fetch[0].Git.URL != in.Source.RepositoryURL {
				t.Errorf("git url = %q, want %q", app.Spec.Fetch[0].Git.URL, in.Source.RepositoryURL)
			}
		})
	}
}

func TestExecute_IsIdempotent(t *testing.T) {
	scheme := newPackageScheme(t)
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := packageProviders(c, scheme)[name]
			in := newPackageIntent()

			for i := 0; i < 2; i++ {
				if _, err := p.Execute(context.Background(), in); err != nil {
					t.Fatalf("Execute #%d: %v", i+1, err)
				}
			}

			var apps kappctrlv1alpha1.AppList
			if err := c.List(context.Background(), &apps); err != nil {
				t.Fatalf("list apps: %v", err)
			}
			if len(apps.Items) != 1 {
				t.Errorf("%d Apps after two runs, want 1", len(apps.Items))
			}
		})
	}
}

func TestExecute_WithoutOwnerUIDSetsNoOwner(t *testing.T) {
	scheme := newPackageScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	in := newPackageIntent()
	in.OwnerUID = ""

	if _, err := NewApplicationProvider(c, scheme, logr.Discard(), nil).Execute(context.Background(), in); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	app, err := getApp(t, c, in.ID)
	if err != nil {
		t.Fatalf("get app: %v", err)
	}
	if len(app.OwnerReferences) != 0 {
		t.Errorf("owner references = %+v, want none", app.OwnerReferences)
	}
}

func TestTeardown_RemovesTheApp(t *testing.T) {
	scheme := newPackageScheme(t)
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := packageProviders(c, scheme)[name]
			in := newPackageIntent()
			if _, err := p.Execute(context.Background(), in); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			// Teardown is idempotent: the second call finds nothing to delete.
			for i := 0; i < 2; i++ {
				if err := p.Teardown(context.Background(), in.ID); err != nil {
					t.Fatalf("Teardown #%d: %v", i+1, err)
				}
			}

			if _, err := getApp(t, c, in.ID); !apierrors.IsNotFound(err) {
				t.Errorf("get app after teardown: err = %v, want not found", err)
			}
		})
	}
}

func TestTeardown_NothingToRemove(t *testing.T) {
	scheme := newPackageScheme(t)
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := packageProviders(c, scheme)[name]
			if err := p.Teardown(context.Background(), domain.PackageID{Namespace: "default", Name: "never-applied"}); err != nil {
				t.Errorf("Teardown: %v", err)
			}
		})
	}
}

// TestExecute_ReportsTheAppPhase covers the phase reported at each stage of
// the App's life. Both providers report what the App says, read the same way
// an observer of the App reads it.
func TestExecute_ReportsTheAppPhase(t *testing.T) {
	scheme := newPackageScheme(t)
	stages := []struct {
		name      string
		status    corev1.ConditionStatus
		message   string
		wantPhase domain.PackagePhase
	}{
		{name: "not reported yet", wantPhase: domain.PackagePhasePending},
		{name: "reconcile failed", status: corev1.ConditionFalse, message: "fetch failed", wantPhase: domain.PackagePhaseFailed},
		{name: "reconcile succeeded", status: corev1.ConditionTrue, wantPhase: domain.PackagePhaseSucceeded},
	}

	for name := range packageProviders(nil, scheme) {
		for _, stage := range stages {
			t.Run(name+"/"+stage.name, func(t *testing.T) {
				c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kappctrlv1alpha1.App{}).Build()
				p := packageProviders(c, scheme)[name]
				in := newPackageIntent()
				if _, err := p.Execute(context.Background(), in); err != nil {
					t.Fatalf("first Execute: %v", err)
				}

				if stage.status != "" {
					app, err := getApp(t, c, in.ID)
					if err != nil {
						t.Fatalf("get app: %v", err)
					}
					app.Status.Conditions = []kappctrlv1alpha1.Condition{{
						Type: kappctrlv1alpha1.ReconcileSucceeded, Status: stage.status, Message: stage.message,
					}}
					if err := c.Status().Update(context.Background(), app); err != nil {
						t.Fatalf("update app status: %v", err)
					}
				}

				res, err := p.Execute(context.Background(), in)
				if err != nil {
					t.Fatalf("Execute: %v", err)
				}
				if res.Phase != stage.wantPhase || res.Message != stage.message {
					t.Errorf("phase = %s message = %q, want %s %q", res.Phase, res.Message, stage.wantPhase, stage.message)
				}
				wantSuccess := stage.wantPhase == domain.PackagePhaseSucceeded
				if res.Success != wantSuccess {
					t.Errorf("success = %v, want %v", res.Success, wantSuccess)
				}
			})
		}
	}
}

// TestApplicationStateFromApp covers how each thing a kapp App can report is
// read. A failed fetch is reported as ReconcileFailed with the detail in
// usefulErrorMessage, and must not be mistaken for an App that is pending.
func TestApplicationStateFromApp(t *testing.T) {
	cond := func(t kappctrlv1alpha1.ConditionType, s corev1.ConditionStatus, msg string) kappctrlv1alpha1.Condition {
		return kappctrlv1alpha1.Condition{Type: t, Status: s, Message: msg}
	}

	tests := []struct {
		name        string
		status      kappctrlv1alpha1.AppStatus
		wantPhase   domain.ApplicationPhase
		wantMessage string
	}{
		{name: "nothing reported", wantPhase: domain.ApplicationPhasePending},
		{
			name:      "reconciling",
			status:    kappctrlv1alpha1.AppStatus{GenericStatus: kappctrlv1alpha1.GenericStatus{Conditions: []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.Reconciling, corev1.ConditionTrue, "")}}},
			wantPhase: domain.ApplicationPhasePending,
		},
		{
			name:      "succeeded",
			status:    kappctrlv1alpha1.AppStatus{GenericStatus: kappctrlv1alpha1.GenericStatus{Conditions: []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileSucceeded, corev1.ConditionTrue, "")}}},
			wantPhase: domain.ApplicationPhaseReady,
		},
		{
			name:        "succeeded condition is false",
			status:      kappctrlv1alpha1.AppStatus{GenericStatus: kappctrlv1alpha1.GenericStatus{Conditions: []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileSucceeded, corev1.ConditionFalse, "not yet")}}},
			wantPhase:   domain.ApplicationPhaseFailed,
			wantMessage: "not yet",
		},
		{
			name: "fetch failed, with the useful message",
			status: kappctrlv1alpha1.AppStatus{GenericStatus: kappctrlv1alpha1.GenericStatus{
				Conditions:         []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileFailed, corev1.ConditionTrue, "Fetching resources: Error")},
				UsefulErrorMessage: "Host key verification failed",
			}},
			wantPhase:   domain.ApplicationPhaseFailed,
			wantMessage: "Host key verification failed",
		},
		{
			name:        "failed without a useful message",
			status:      kappctrlv1alpha1.AppStatus{GenericStatus: kappctrlv1alpha1.GenericStatus{Conditions: []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileFailed, corev1.ConditionTrue, "Fetching resources: Error")}}},
			wantPhase:   domain.ApplicationPhaseFailed,
			wantMessage: "Fetching resources: Error",
		},
		{
			name: "deploy error overrides the message",
			status: kappctrlv1alpha1.AppStatus{
				GenericStatus: kappctrlv1alpha1.GenericStatus{Conditions: []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileFailed, corev1.ConditionTrue, "Deploying: Error")}},
				Deploy:        &kappctrlv1alpha1.AppStatusDeploy{Finished: true, ExitCode: 1, Error: "kapp: resource rejected"},
			},
			wantPhase:   domain.ApplicationPhaseFailed,
			wantMessage: "kapp: resource rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &kappctrlv1alpha1.App{Status: tt.status}
			app.Name, app.Namespace = "app-package", "default"

			state := ApplicationStateFromApp(app)
			if state.Phase != tt.wantPhase || state.Message != tt.wantMessage {
				t.Errorf("phase = %s message = %q, want %s %q", state.Phase, state.Message, tt.wantPhase, tt.wantMessage)
			}
			if state.Name != "app-package" || state.Namespace != "default" {
				t.Errorf("identity = %s/%s", state.Namespace, state.Name)
			}
		})
	}
}
