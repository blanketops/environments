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

	"github.com/go-logr/logr"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/utils"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"

	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/client-go/tools/events"
)

type BuildpacksProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

func NewBuildpacksProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *BuildpacksProvider {
	return &BuildpacksProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

//
// -----------------------------------------------------------------------------
// Shipwright BUILD Spec
// -----------------------------------------------------------------------------
//

func (p *BuildpacksProvider) CreateBuildSpec(
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
			// Timeout intentionally omitted (provider-level concern)
		},
	}

	return ship, nil
}

//
// -----------------------------------------------------------------------------
// Shipwright BUILDRUN Spec
// -----------------------------------------------------------------------------
//

func (p *BuildpacksProvider) CreateBuildRunSpec(
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

//
// -----------------------------------------------------------------------------
// Provider Run()
// -----------------------------------------------------------------------------
//

func (p *BuildpacksProvider) Run(
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
	// 1. Create / Update Shipwright Build
	// ------------------------------------------------
	shipBuild, err := p.CreateBuildSpec(spec, build)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	if err := controllerutil.SetControllerReference(
		build.Build,
		shipBuild,
		p.Scheme,
	); err != nil {
		res.Message = err.Error()
		return res, err
	}

	foundBuild := &shipwrightv1alpha1.Build{}
	getErr := p.Client.Get(
		ctx,
		client.ObjectKeyFromObject(shipBuild),
		foundBuild,
	)

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
	// 2. Create BuildRun (idempotent)
	// ------------------------------------------------
	tc := ExtractTriggerContext(build.Build)

	hash, err := utils.ComputeExecutionHash(
		build.Spec.ToBuildContract(),
		tc,
	)

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

	if err := controllerutil.SetControllerReference(
		build.Build,
		buildRun,
		p.Scheme,
	); err != nil {
		res.Message = err.Error()
		return res, err
	}

	foundRun := &shipwrightv1alpha1.BuildRun{}
	getErr = p.Client.Get(
		ctx,
		client.ObjectKeyFromObject(buildRun),
		foundRun,
	)

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
	// SUCCESS (execution triggered)
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
