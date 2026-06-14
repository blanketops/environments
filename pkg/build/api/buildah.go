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

/*
Package api implements the build provider layer for the BlanketOps
Environments build domain.

This file owns the Buildah provider — the concrete implementation that
translates a resolved BlanketOps Build contract into Shipwright resources
and dispatches execution.

The provider sits below the application service layer and above the
Shipwright API. It is responsible for:
  - Constructing Shipwright Build and BuildRun specs from the resolved contract.
  - Applying owner references so Shipwright resources are garbage-collected
    with the parent Build CR.
  - Ensuring idempotent upsert of the Shipwright Build (create or update).
  - Creating the BuildRun only when no run for the current execution hash
    exists, preventing duplicate builds on re-reconciliation.

The provider does NOT wait for the BuildRun to complete. Completion is
observed asynchronously by the buildrun observer (internal/controller/observers/buildrun).
Run() returns Triggered=true, Success=false to signal intent dispatch — the
final outcome is written to the Build CR by the observer.
*/
package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/utils"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
)

// BuildahProvider orchestrates Shipwright Build and BuildRun resources on
// behalf of the BlanketOps build domain. It is the only component in the
// platform that writes Shipwright API objects directly.
type BuildahProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	// Recorder is optional — may be nil. No events are emitted at the
	// provider layer; events are owned by the domain and observer layers.
	Recorder events.EventRecorder
}

// NewBuildahProvider constructs a BuildahProvider with the given dependencies.
// rec may be nil — the provider does not emit events directly.
func NewBuildahProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *BuildahProvider {
	return &BuildahProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

// -----------------------------------------------------------------------------
// Shipwright Build spec construction
// -----------------------------------------------------------------------------

// CreateBuildSpec translates a domain.BuildSpec and resolved Build contract
// into a Shipwright Build object. Validates that SourceURL and Image are
// present — both are required for Shipwright to accept the Build.
//
// The returned Build has labels for domain ownership tracking
// ("build.blanketops.dev/name") but no owner reference — that is applied
// by the caller via controllerutil.SetControllerReference before creation.
//
// Timeout is intentionally omitted from the Shipwright BuildSpec — timeout
// policy is enforced at the BuildRun level by Shipwright's own machinery.
func (p *BuildahProvider) CreateBuildSpec(
	spec domain.BuildSpec,
	build *buildResolution.ResolvedBuild,
) (*shipwrightv1alpha1.Build, error) {
	if spec.SourceURL == "" {
		return nil, fmt.Errorf("missing source URL")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("missing image target")
	}

	strategyKind := shipwrightv1alpha1.BuildStrategyKind(spec.StrategyKind)

	ship := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      build.Build.Name,
			Namespace: build.Build.Namespace,
			Labels: map[string]string{
				"build.blanketops.dev/name": build.Build.Name,
				"strategy":                  spec.StrategyName,
			},
		},
		Spec: shipwrightv1alpha1.BuildSpec{
			Source: shipwrightv1alpha1.Source{
				URL:        &spec.SourceURL,
				ContextDir: &spec.ContextDir,
				Revision:   &spec.Revision,
				Credentials: &corev1.LocalObjectReference{
					Name: spec.CloneSecret,
				},
			},
			Strategy: shipwrightv1alpha1.Strategy{
				Name: spec.StrategyName,
				Kind: &strategyKind,
			},
			Output: shipwrightv1alpha1.Image{
				Image: spec.Image,
				Credentials: &corev1.LocalObjectReference{
					Name: spec.ServiceAccountSecret,
				},
			},
			// Timeout intentionally omitted — enforced at BuildRun level.
		},
	}

	return ship, nil
}

// -----------------------------------------------------------------------------
// Shipwright BuildRun spec construction
// -----------------------------------------------------------------------------

// CreateBuildRunSpec constructs a Shipwright BuildRun for the given resolved
// Build and execution hash. The run name is derived from the Build name and
// a short hash suffix, ensuring a unique run per execution identity.
//
// Labels carry the execution identity for the buildrun observer to resolve
// the owning Build CR and for retry counting (list by build name + hash).
// The full hash is preserved in an annotation for audit purposes.
func (p *BuildahProvider) CreateBuildRunSpec(
	build *buildResolution.ResolvedBuild,
	shipwrightBuild *shipwrightv1alpha1.Build,
	fullHash string,
) *shipwrightv1alpha1.BuildRun {
	short := utils.ShortHash(fullHash)
	name := fmt.Sprintf("%s-%s", build.Build.Name, short)

	return &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: build.Build.Namespace,
			Labels: map[string]string{
				"build.blanketops.dev/name": build.Build.Name,
				"build-hash":                short,
				"shipwright-build":          shipwrightBuild.Name,
			},
			Annotations: map[string]string{
				// Full hash preserved for audit; short hash used for run
				// naming and label-based lookups.
				"build.blanketops.dev/hash": fullHash,
			},
		},
		Spec: shipwrightv1alpha1.BuildRunSpec{
			BuildRef: &shipwrightv1alpha1.BuildRef{
				Name: shipwrightBuild.Name,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Provider execution
// -----------------------------------------------------------------------------

// Run orchestrates the full Shipwright dispatch pipeline for the given
// resolved Build and spec. It upserts the Shipwright Build and creates the
// BuildRun, then returns immediately with Triggered=true.
//
// Run does NOT block on BuildRun completion. The buildrun observer
// (internal/controller/observers/buildrun) watches for terminal state and
// writes the final outcome back to the Build CR asynchronously.
//
// Idempotency: the Shipwright Build is always upserted. The BuildRun is only
// created if no run for the current execution hash exists — a hash collision
// (same spec + same trigger context) is treated as "already triggered" and
// the existing run is reused.
func (p *BuildahProvider) Run(
	ctx context.Context,
	build *buildResolution.ResolvedBuild,
	spec domain.BuildSpec,
) (domain.BuildResult, error) {
	res := domain.BuildResult{
		Success: false,
		Message: "",
	}

	p.Log.Info(
		"provider.run: starting orchestration",
		"build", client.ObjectKeyFromObject(build.Build).String(),
		"strategy", spec.StrategyName,
	)

	// ------------------------------------------------
	// Stage 1: Upsert the Shipwright Build.
	//
	// The Build object declares the strategy, source, and output image.
	// It is stable across retries — only the BuildRun varies per execution.
	// ------------------------------------------------
	shipBuild, err := p.CreateBuildSpec(spec, build)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	if err := controllerutil.SetControllerReference(build.Build, shipBuild, p.Scheme); err != nil {
		res.Message = err.Error()
		return res, err
	}

	foundBuild := &shipwrightv1alpha1.Build{}
	getErr := p.Client.Get(ctx, client.ObjectKeyFromObject(shipBuild), foundBuild)

	if apierrors.IsNotFound(getErr) {
		if err := p.Client.Create(ctx, shipBuild); err != nil {
			res.Message = err.Error()
			return res, err
		}
	} else if getErr == nil {
		shipBuild.ResourceVersion = foundBuild.ResourceVersion
		if err := p.Client.Update(ctx, shipBuild); err != nil {
			res.Message = err.Error()
			return res, err
		}
	} else {
		res.Message = getErr.Error()
		return res, getErr
	}

	// ------------------------------------------------
	// Stage 2: Create the BuildRun (idempotent by hash).
	//
	// The execution hash combines the resolved spec and trigger context
	// (commit SHA, retry attempt, trigger type). A run already exists for
	// this hash if and only if this exact execution was already dispatched —
	// in that case we skip creation and return Triggered=true against the
	// existing run.
	// ------------------------------------------------
	tc := ExtractTriggerContext(build.Build)
	hash, err := utils.ComputeExecutionHash(build.Spec.ToBuildContract(), tc)

	p.Log.Info(
		"execution identity",
		"retryAttempt", tc.RetryAttempt,
		"triggerType", tc.Type,
	)

	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	buildRun := p.CreateBuildRunSpec(build, shipBuild, hash)

	if err := controllerutil.SetControllerReference(build.Build, buildRun, p.Scheme); err != nil {
		res.Message = err.Error()
		return res, err
	}

	foundRun := &shipwrightv1alpha1.BuildRun{}
	getErr = p.Client.Get(ctx, client.ObjectKeyFromObject(buildRun), foundRun)

	if apierrors.IsNotFound(getErr) {
		if err := p.Client.Create(ctx, buildRun); err != nil {
			res.Message = err.Error()
			return res, err
		}
	} else if getErr != nil {
		res.Message = getErr.Error()
		return res, getErr
	}

	// ------------------------------------------------
	// Execution dispatched — return intent result.
	//
	// Success=false here is intentional: the build has not succeeded yet,
	// it has only been triggered. The observer sets Success=true when the
	// BuildRun completes.
	// ------------------------------------------------
	res.Success = false
	res.Triggered = true
	res.ExecutionRef = buildRun.Name
	res.Message = "BuildRun created"

	p.Log.Info(
		"provider.run: orchestration complete",
		"build", build.Build.Name,
		"buildRun", buildRun.Name,
		"image", spec.Image,
	)

	return res, nil
}