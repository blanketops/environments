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
	"errors"
	"testing"

	kappctrlv1alpha1 "carvel.dev/kapp-controller/pkg/apis/kappctrl/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

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
		Source:      domain.PackageSource{RepositoryURL: "git@github.com:example-org/packages.git", Path: "manifests"},
		ResolvedRef: "origin/main",
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

// kapp-controller holds an App with a finalizer while it removes what the App
// deployed. Until the App is gone Teardown reports that it is in progress, so
// the caller does not take away the service account that removal runs as.
func TestTeardown_InProgressWhileTheAppIsBeingDeleted(t *testing.T) {
	scheme := newPackageScheme(t)
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			in := newPackageIntent()
			held := &kappctrlv1alpha1.App{}
			held.Name, held.Namespace = in.ID.Name, in.ID.Namespace
			held.Finalizers = []string{"finalizers.kapp-ctrl.k14s.io/delete"}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(held).Build()
			p := packageProviders(c, scheme)[name]

			// Asked twice: the App is still there both times.
			for i := 0; i < 2; i++ {
				if err := p.Teardown(context.Background(), in.ID); !errors.Is(err, domain.ErrTeardownInProgress) {
					t.Fatalf("Teardown #%d = %v, want ErrTeardownInProgress", i+1, err)
				}
			}

			// kapp-controller finishes and releases the App.
			app, err := getApp(t, c, in.ID)
			if err != nil {
				t.Fatalf("get app: %v", err)
			}
			if app.DeletionTimestamp.IsZero() {
				t.Fatal("the App should be marked for deletion")
			}
			app.Finalizers = nil
			if err := c.Update(context.Background(), app); err != nil {
				t.Fatalf("release the App: %v", err)
			}

			if err := p.Teardown(context.Background(), in.ID); err != nil {
				t.Errorf("Teardown once the App is gone: %v", err)
			}
		})
	}
}

// A failure to read the App back is reported as it is, not as progress.
func TestTeardown_ErrorReadingTheApp(t *testing.T) {
	scheme := newPackageScheme(t)
	boom := errors.New("boom")
	for name := range packageProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return boom
				},
			}).Build()
			p := packageProviders(c, scheme)[name]

			err := p.Teardown(context.Background(), domain.PackageID{Namespace: "default", Name: "app-package"})
			if !errors.Is(err, boom) || errors.Is(err, domain.ErrTeardownInProgress) {
				t.Errorf("Teardown = %v, want the read error and not ErrTeardownInProgress", err)
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
			name: "deploy failed, with the useful message",
			status: kappctrlv1alpha1.AppStatus{
				GenericStatus: kappctrlv1alpha1.GenericStatus{
					Conditions:         []kappctrlv1alpha1.Condition{cond(kappctrlv1alpha1.ReconcileFailed, corev1.ConditionTrue, "Deploying: Error (see .status.usefulErrorMessage for details)")},
					UsefulErrorMessage: "kapp: Error: configmaps is forbidden",
				},
				Deploy: &kappctrlv1alpha1.AppStatusDeploy{Finished: true, ExitCode: 1, Error: "Deploying: Error (see .status.usefulErrorMessage for details)"},
			},
			wantPhase:   domain.ApplicationPhaseFailed,
			wantMessage: "kapp: Error: configmaps is forbidden",
		},
		{
			name: "deploy error is the message when there is no useful one",
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

// TestBuildKappApplication_Fetch covers what the App is told about the
// package repository. Credentials are referenced only when the contract
// declares them.
func TestBuildKappApplication_Fetch(t *testing.T) {
	tests := []struct {
		name        string
		credentials string
		wantSecret  string
	}{
		{name: "private repository", credentials: "packages-creds", wantSecret: "packages-creds"},
		{name: "public repository"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := newPackageIntent()
			in.Source.CredentialsSecret = tt.credentials

			app, err := BuildKappApplication(in)
			if err != nil {
				t.Fatalf("BuildKappApplication: %v", err)
			}
			if len(app.Spec.Fetch) != 1 || app.Spec.Fetch[0].Git == nil {
				t.Fatalf("fetch = %+v, want one git source", app.Spec.Fetch)
			}
			git := app.Spec.Fetch[0].Git
			if git.URL != in.Source.RepositoryURL {
				t.Errorf("url = %q, want %q", git.URL, in.Source.RepositoryURL)
			}
			if git.Ref != "origin/main" || git.SubPath != "manifests" {
				t.Errorf("ref = %q subPath = %q, want origin/main and manifests", git.Ref, git.SubPath)
			}
			switch {
			case tt.wantSecret == "" && git.SecretRef != nil:
				t.Errorf("secretRef = %+v, want none", git.SecretRef)
			case tt.wantSecret != "" && (git.SecretRef == nil || git.SecretRef.Name != tt.wantSecret):
				t.Errorf("secretRef = %+v, want %q", git.SecretRef, tt.wantSecret)
			}
		})
	}
}

// TestBuildKappApplication_ServiceAccount: kapp-controller does not deploy
// without an identity, so the App always names the Package's ServiceAccount.
func TestBuildKappApplication_ServiceAccount(t *testing.T) {
	in := newPackageIntent()
	app, err := BuildKappApplication(in)
	if err != nil {
		t.Fatalf("BuildKappApplication: %v", err)
	}
	if app.Spec.ServiceAccountName != "app-package-package" {
		t.Errorf("serviceAccountName = %q, want app-package-package", app.Spec.ServiceAccountName)
	}
	if app.Spec.ServiceAccountName != in.ID.ServiceAccountName() {
		t.Errorf("serviceAccountName = %q, want the name the domain decides, %q", app.Spec.ServiceAccountName, in.ID.ServiceAccountName())
	}
}

func TestBuildKappApplication_TemplatesWithYttAndDeploysWithKapp(t *testing.T) {
	app, err := BuildKappApplication(newPackageIntent())
	if err != nil {
		t.Fatalf("BuildKappApplication: %v", err)
	}
	if len(app.Spec.Template) != 1 || app.Spec.Template[0].Ytt == nil {
		t.Errorf("template = %+v, want one ytt stage", app.Spec.Template)
	}
	if len(app.Spec.Deploy) != 1 || app.Spec.Deploy[0].Kapp == nil {
		t.Errorf("deploy = %+v, want one kapp stage", app.Spec.Deploy)
	}
}
