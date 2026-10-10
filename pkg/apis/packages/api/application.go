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
	"fmt"
	"time"

	kappctrlv1alpha1 "carvel.dev/kapp-controller/pkg/apis/kappctrl/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"
	"github.com/blanketops/environments/pkg/intent/package"
)

// ApplicationProvider executes a Package via a kapp-controller App. See
// PackageProvider for the newer implementation with corrected phase
// mapping.
type ApplicationProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder // optional
}

// Compile-time contract check
var _ Provider = (*ApplicationProvider)(nil)

// NewApplicationProvider constructs an ApplicationProvider.
func NewApplicationProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *ApplicationProvider {
	return &ApplicationProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

// Execute builds the kapp-controller App for the intent and applies it.
func (p *ApplicationProvider) Execute(
	ctx context.Context,
	intent *intent.PackageIntent,
) (*domain.PackageResult, error) {

	start := time.Now()

	log := p.Log.WithValues(
		"package", intent.ID.Name,
		"namespace", intent.ID.Namespace,
	)

	log.Info("Executing package via kapp App")

	// ------------------------------------------------------------
	// 1. Build kapp App spec
	// ------------------------------------------------------------
	app, err := BuildKappApplication(intent)
	if err != nil {
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 2. Apply kapp App (SSA)
	// ------------------------------------------------------------
	if err := ApplyApplication(ctx, p.Client, app); err != nil {
		log.Error(err, "Failed to apply kapp App")
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 3. Observe kapp App status
	// ------------------------------------------------------------
	state, err := p.ObserveApplication(
		ctx,
		app.Namespace,
		app.Name,
	)
	if err != nil {
		log.Error(err, "Failed to observe kapp App")
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 4. Build domain result
	//
	// The result says what the App reported at this moment. It is usually
	// still pending right after the apply; the outcome that follows is
	// recorded by whoever observes the App, reading it the same way.
	// ------------------------------------------------------------
	result := PackageResultFromApplicationState(state)
	result.StartedAt = start
	result.FinishedAt = time.Now()

	return result, nil
}

// ApplyApplication server-side applies the App with the blanketops-packages
// field manager, forcing ownership of any conflicting fields.
func ApplyApplication(
	ctx context.Context,
	c client.Client,
	app *kappctrlv1alpha1.App,
) error {

	return c.Patch(
		ctx,
		app,
		client.Apply, //nolint:staticcheck
		&client.PatchOptions{FieldManager: "blanketops-packages", Force: ptr.To(true)},
	)
}

// BuildKappApplication constructs the kapp-controller App object for a
// package intent. Pure function: no side effects, no cluster access.
func BuildKappApplication(
	intent *intent.PackageIntent,
) (*kappctrlv1alpha1.App, error) {

	return &kappctrlv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kappctrlv1alpha1.SchemeGroupVersion.String(),
			Kind:       "App",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            intent.ID.Name,
			Namespace:       intent.ID.Namespace,
			Labels:          intent.Labels,
			OwnerReferences: ownerReferences(intent),
		},
		Spec: kappctrlv1alpha1.AppSpec{
			// The identity kapp-controller deploys as. It is provisioned
			// as a prerequisite of the Package, not here.
			ServiceAccountName: intent.ID.ServiceAccountName(),

			// Controller-driven reconciliation
			SyncPeriod: &metav1.Duration{Duration: 0},

			Fetch: []kappctrlv1alpha1.AppFetch{
				{
					Git: &kappctrlv1alpha1.AppFetchGit{
						URL:       intent.Source.RepositoryURL,
						Ref:       intent.ResolvedRef,
						SecretRef: fetchSecretRef(intent.Source.CredentialsSecret),
						SubPath:   intent.Source.Path,
					},
				},
			},

			// The package repository holds plain manifests. ytt passes
			// them through and gives the repository a place to add
			// overlays later.
			Template: []kappctrlv1alpha1.AppTemplate{
				{
					Ytt: &kappctrlv1alpha1.AppTemplateYtt{},
				},
			},

			Deploy: []kappctrlv1alpha1.AppDeploy{
				{
					Kapp: &kappctrlv1alpha1.AppDeployKapp{},
				},
			},
		},
	}, nil
}

// ObserveApplication fetches the current state of the named App.
func (p *ApplicationProvider) ObserveApplication(
	ctx context.Context,
	namespace,
	name string,
) (*domain.ApplicationState, error) {

	var app kappctrlv1alpha1.App
	if err := p.Client.Get(
		ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to get kapp App %s/%s: %w",
			namespace,
			name,
			err,
		)
	}

	return ApplicationStateFromApp(&app), nil
}

// ApplicationStateFromApp derives an ApplicationState from what a kapp App
// reports about itself. It is the one place the App's conditions are
// interpreted, shared by the providers and by observers of the App.
//
// kapp-controller reports success as ReconcileSucceeded and failure as a
// separate ReconcileFailed condition, with the detail in
// status.usefulErrorMessage. An App that has reported neither is pending.
func ApplicationStateFromApp(app *kappctrlv1alpha1.App) *domain.ApplicationState {
	state := &domain.ApplicationState{
		Name:      app.Name,
		Namespace: app.Namespace,
		Phase:     domain.ApplicationPhasePending,
	}

	// --------------------------------------------------------
	// Phase resolution (ReconcileSucceeded and ReconcileFailed)
	// --------------------------------------------------------
	for _, cond := range app.Status.Conditions {
		switch cond.Type {
		case kappctrlv1alpha1.ReconcileSucceeded:
			switch cond.Status {
			case corev1.ConditionTrue:
				state.Phase = domain.ApplicationPhaseReady

			case corev1.ConditionFalse:
				state.Phase = domain.ApplicationPhaseFailed
				state.Message = cond.Message

			case corev1.ConditionUnknown:
				state.Phase = domain.ApplicationPhasePending
			}

		case kappctrlv1alpha1.ReconcileFailed:
			if cond.Status == corev1.ConditionTrue {
				state.Phase = domain.ApplicationPhaseFailed
				state.Message = cond.Message
				if app.Status.UsefulErrorMessage != "" {
					state.Message = app.Status.UsefulErrorMessage
				}
			}
		}
	}

	// --------------------------------------------------------
	// Deploy execution details
	// --------------------------------------------------------
	if d := app.Status.Deploy; d != nil {

		if !d.StartedAt.IsZero() {
			t := d.StartedAt.Time
			state.DeployStartedAt = &t
		}

		if !d.UpdatedAt.IsZero() {
			t := d.UpdatedAt.Time
			state.DeployUpdatedAt = &t
		}

		state.DeployFinished = d.Finished

		if d.ExitCode != 0 {
			code := d.ExitCode
			state.DeployExitCode = &code
		}

		// kapp-controller's deploy error is a pointer to the detail
		// ("see .status.usefulErrorMessage") whenever it has one, so it
		// is used only when there is no useful message to report.
		if d.Error != "" && app.Status.UsefulErrorMessage == "" {
			state.Message = d.Error
		}
	}

	return state
}

// failedResult builds a failed PackageResult stamped with start and err's message.
func failedResult(start time.Time, err error) *domain.PackageResult {
	return &domain.PackageResult{
		Success:    false,
		Phase:      domain.PackagePhaseFailed,
		Message:    err.Error(),
		StartedAt:  start,
		FinishedAt: time.Now(),
	}
}

// fetchSecretRef references the Secret kapp-controller authenticates the
// fetch with. Returns nil when the contract declares no credentials, so a
// public repository gets no reference at all.
func fetchSecretRef(name string) *kappctrlv1alpha1.AppFetchLocalRef {
	if name == "" {
		return nil
	}
	return &kappctrlv1alpha1.AppFetchLocalRef{Name: name}
}

// ownerReferences makes the Package CR the controlling owner of an object
// created for it. Returns nil when the intent carries no owner UID.
func ownerReferences(intent *intent.PackageIntent) []metav1.OwnerReference {
	if intent.OwnerUID == "" {
		return nil
	}
	return []metav1.OwnerReference{{
		APIVersion:         environmentv1alpha1.GroupVersion.String(),
		Kind:               "Package",
		Name:               intent.ID.Name,
		UID:                intent.OwnerUID,
		Controller:         ptr.To(true),
		BlockOwnerDeletion: ptr.To(true),
	}}
}

// Teardown deletes the kapp App created for the Package. kapp-controller
// then removes the resources the App deployed. Idempotent — a missing App is
// not an error.
func (p *ApplicationProvider) Teardown(ctx context.Context, id domain.PackageID) error {
	if err := DeleteApplication(ctx, p.Client, id); err != nil {
		return err
	}
	p.Log.Info("provider.teardown: complete", "package", id.Name, "namespace", id.Namespace)
	return nil
}

// DeleteApplication deletes the kapp App named after the Package. A missing
// App is not an error.
//
// Deleting an App is not immediate: kapp-controller first removes what the
// App deployed, as the App's service account. While the App still exists
// this returns domain.ErrTeardownInProgress, so the caller keeps that service
// account until a later call finds the App gone.
func DeleteApplication(ctx context.Context, c client.Client, id domain.PackageID) error {
	app := &kappctrlv1alpha1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id.Name,
			Namespace: id.Namespace,
		},
	}
	if err := c.Delete(ctx, app); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete kapp app %s/%s: %w", id.Namespace, id.Name, err)
	}

	err := c.Get(ctx, types.NamespacedName{Namespace: id.Namespace, Name: id.Name}, app)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get kapp app %s/%s: %w", id.Namespace, id.Name, err)
	}
	return fmt.Errorf("kapp app %s/%s is still deleting: %w", id.Namespace, id.Name, domain.ErrTeardownInProgress)
}
