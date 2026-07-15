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
	"time"

	kappctrlv1alpha1 "carvel.dev/kapp-controller/pkg/apis/kappctrl/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"
	"github.com/blanketops/environments/pkg/apis/packages/intent"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PackageProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder // optional; may be nil
}

func NewPackageProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, rec events.EventRecorder) *PackageProvider {
	// recorder may be nil in unit tests; keep it optional
	return &PackageProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

func (p *PackageProvider) Execute(
	ctx context.Context,
	intent *intent.PackageIntent,
) (*domain.PackageResult, error) {

	start := time.Now()

	log := p.Log.WithValues(
		"package", intent.ID.Name,
		"namespace", intent.ID.Namespace,
	)

	log.Info("Executing package via kapp App")

	// 1. Build
	app, err := BuildKappApplication(intent)
	if err != nil {
		return failedResult(start, err), err
	}

	// 2. Apply
	if err := ApplyApplication(ctx, p.Client, app); err != nil {
		log.Error(err, "Failed to apply kapp App")
		return failedResult(start, err), err
	}

	// 3. Observe
	state, err := p.ObserveApplication(ctx, app.Namespace, app.Name)
	if err != nil {
		log.Error(err, "Failed to observe kapp App")
		return failedResult(start, err), err
	}

	// 4. Translate → domain
	result := PackageResultFromApplicationState(state)
	result.StartedAt = start
	result.FinishedAt = time.Now()

	return result, nil
}

func PackageResultFromApplicationState(
	state *domain.ApplicationState,
) *domain.PackageResult {

	result := &domain.PackageResult{
		Success: state.Phase == domain.ApplicationPhaseReady,
		Message: state.Message,
		Kapp: domain.KappResult{
			Name:      state.Name,
			Namespace: state.Namespace,
		},
	}

	// Phase mapping (EXPLICIT)
	switch state.Phase {
	case domain.ApplicationPhaseReady:
		result.Phase = domain.PackagePhaseSucceeded

	case domain.ApplicationPhaseFailed:
		result.Phase = domain.PackagePhaseFailed

	case domain.ApplicationPhasePending:
		result.Phase = domain.PackagePhasePending

	default:
		result.Phase = domain.PackagePhaseUnknown
	}

	// Execution details (optional but powerful)
	result.Kapp.DeployStartedAt = state.DeployStartedAt
	result.Kapp.DeployUpdatedAt = state.DeployUpdatedAt
	result.Kapp.DeployFinished = state.DeployFinished
	result.Kapp.DeployExitCode = state.DeployExitCode

	return result
}

func (p *PackageProvider) ObserveApplication(
	ctx context.Context,
	namespace,
	name string,
) (*domain.ApplicationState, error) {

	var app kappctrlv1alpha1.App
	if err := p.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		return nil, err
	}

	state := &domain.ApplicationState{
		Name:      app.Name,
		Namespace: app.Namespace,
		Phase:     domain.ApplicationPhasePending,
	}

	// ------------------------------------------------------------
	// Phase resolution (ReconcileSucceeded is authoritative)
	// ------------------------------------------------------------
	for _, cond := range app.Status.Conditions {
		if cond.Type != "ReconcileSucceeded" {
			continue
		}

		switch cond.Status {
		case corev1.ConditionTrue:
			state.Phase = domain.ApplicationPhaseReady

		case corev1.ConditionFalse:
			state.Phase = domain.ApplicationPhaseFailed
			state.Message = cond.Message

		case corev1.ConditionUnknown:
			state.Phase = domain.ApplicationPhasePending
		}
	}

	// ------------------------------------------------------------
	// Deploy execution details (truth from kapp)
	// ------------------------------------------------------------
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

		if d.Error != "" {
			state.Message = d.Error
		}
	}

	return state, nil
}
