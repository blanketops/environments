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

This file owns the Kaniko provider — structurally identical to the Kaniko
provider but registered under the "kaniko" strategy name. Shipwright selects
the ClusterBuildStrategy at run time; the provider layer constructs the spec
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
	"strings"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/utils"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
	githubEventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// KanikoProvider orchestrates Shipwright Build and BuildRun resources on
// behalf of the BlanketOps build domain. It is the only component in the
// platform that writes Shipwright API objects directly.
type KanikoProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

// NewKanikoProvider constructs a KanikoProvider with the given dependencies.
// rec may be nil — the provider does not emit events directly.
func NewKanikoProvider(client client.Client, scheme *runtime.Scheme, log logr.Logger, recorder events.EventRecorder) *KanikoProvider {
	return &KanikoProvider{Client: client, Scheme: scheme, Log: log, Recorder: recorder}
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
func (p *KanikoProvider) CreateBuildSpec(spec domain.BuildSpec, build *buildResolution.ResolvedBuild) (*shipwrightv1alpha1.Build, error) {
	if spec.SourceURL == "" {
		return nil, fmt.Errorf("missing source URL")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("missing image target")
	}

	strategyKind := shipwrightv1alpha1.BuildStrategyKind(spec.StrategyKind)

	return &shipwrightv1alpha1.Build{
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
	}, nil
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
func (p *KanikoProvider) CreateBuildRunSpec(build *buildResolution.ResolvedBuild, shipwrightBuild *shipwrightv1alpha1.Build, fullHash string) *shipwrightv1alpha1.BuildRun {
	short := utils.ShortHash(fullHash)
	return &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", build.Build.Name, short),
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
			BuildRef: &shipwrightv1alpha1.BuildRef{Name: shipwrightBuild.Name},
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
func (p *KanikoProvider) Run(ctx context.Context, build *buildResolution.ResolvedBuild, spec domain.BuildSpec) (domain.BuildResult, error) {
	res := domain.BuildResult{
		Success: false,
		Message: "",
	}

	p.Log.Info("provider.run: starting orchestration", "build", client.ObjectKeyFromObject(build.Build).String(), "strategy", spec.StrategyName)

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

	if err := patchTriggerSHA(ctx, p.Client, build.Build, spec.SourceURL); err != nil {
		res.Message = err.Error()
		return res, err
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

	p.Log.Info("execution identity", "retryAttempt", tc.RetryAttempt, "triggerType", tc.Type)

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
	res.Triggered = true
	res.ExecutionRef = buildRun.Name
	res.Message = "BuildRun " + buildRun.Name + "created"

	p.Log.Info("provider.run: orchestration complete", "build", build.Build.Name, "buildRun", buildRun.Name, "image", spec.Image)
	return res, nil
}

// patchTriggerSHA looks up the most recent push GitHubEvent for this Build's
// repo and patches the discovered SHA onto the Build's annotations so
// ExtractTriggerContext can read it. Shared across all build providers.
// No-op if nothing matches (manual trigger case).
func patchTriggerSHA(ctx context.Context, c client.Client, build *buildv1.Build, sourceURL string) error {
	var events eventsv1alpha1.GitHubEventList
	if err := c.List(ctx, &events,
		client.InNamespace(build.Namespace),
		client.MatchingLabels{"events.blanketops.dev/repo": RepoSlug(sourceURL)},
	); err != nil {
		return err
	}

	var latest *eventsv1alpha1.GitHubEvent
	var latestSpec *githubEventResolution.ResolvedGitHubEventSpec

	for i := range events.Items {
		ev := &events.Items[i]
		resolved, err := githubEventResolution.ResolveGitHubEvent(ev)
		if err != nil {
			continue // malformed contract — skip rather than fail the whole lookup
		}
		if resolved.Spec.EventType != "push" {
			continue
		}
		if latest == nil || ev.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = ev
			latestSpec = resolved.Spec
		}
	}

	if latest == nil {
		return nil
	}

	original := build.DeepCopy()
	if build.Annotations == nil {
		build.Annotations = map[string]string{}
	}
	build.Annotations["build.blanketops.dev/trigger-type"] = "push"
	build.Annotations["build.blanketops.dev/trigger-ref"] = latestSpec.Ref
	build.Annotations["build.blanketops.dev/trigger-sha"] = latestSpec.CommitSHA

	return c.Patch(ctx, build, client.MergeFrom(original))
}

// repoSlug normalizes a Build's source URL into the same label format the
// Sensor writes onto GitHubEvent (events.blanketops.dev/repo).
func RepoSlug(sourceURL string) string {
	s := strings.TrimPrefix(sourceURL, "https://github.com/")
	s = strings.TrimSuffix(s, ".git")
	return strings.ReplaceAll(s, "/", "-")
}
