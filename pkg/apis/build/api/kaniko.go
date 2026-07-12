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

	"github.com/BlanketOps/environments/pkg/utils"
	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildv1 "github.com/BlanketOps/environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/BlanketOps/environments-api/api/events/v1alpha1"
	"github.com/BlanketOps/environments/pkg/apis/build/domain"
	buildResolution "github.com/BlanketOps/environments/resolution/build"
	githubeventResolution "github.com/BlanketOps/environments/resolution/githubevent"
)

const (
	triggerTypeAnnotation   = "build.blanketops.dev/trigger-type"
	triggerRefAnnotation    = "build.blanketops.dev/trigger-ref"
	triggerSHAAnnotation    = "build.blanketops.dev/trigger-sha"
	triggerSourceAnnotation = "build.blanketops.dev/trigger-source"
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

	// ------------------------------------------------
	// Derive image tag and git revision from trigger annotations.
	// If a webhook delivered a commit SHA, build that exact commit
	// and tag the image with it instead of the static spec value.
	// ------------------------------------------------
	image := spec.Image
	revision := spec.Revision
	if sha := build.Build.Annotations[triggerSHAAnnotation]; sha != "" {
		image = replaceImageTag(image, sha)
		revision = sha
	}

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
				Revision:   &revision,
				Credentials: &corev1.LocalObjectReference{
					Name: spec.CloneSecret,
				},
			},
			Strategy: shipwrightv1alpha1.Strategy{
				Name: spec.StrategyName,
				Kind: &strategyKind,
			},
			Output: shipwrightv1alpha1.Image{
				Image: image,
				Credentials: &corev1.LocalObjectReference{
					Name: spec.ServiceAccountSecret,
				},
			},
		},
	}, nil
}

// replaceImageTag swaps the last :tag segment of an image reference.
// Safe for registry paths like ghcr.io/owner/repo:tag.
func replaceImageTag(image, tag string) string {
	if i := strings.LastIndex(image, ":"); i != -1 && strings.Contains(image[:i], "/") {
		return image[:i] + ":" + tag
	}
	return image + ":" + tag
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

	if err := PatchBuildTriggerFromGitHubEvent(ctx, p.Client, build.Build); err != nil {
		res.Message = err.Error()
		return res, err
	}

	// ------------------------------------------------
	// Stage 2: Create the BuildRun (idempotent by hash).
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
	// ------------------------------------------------
	res.Triggered = true
	res.ExecutionRef = buildRun.Name
	res.Message = "BuildRun " + buildRun.Name + "created"

	p.Log.Info("provider.run: orchestration complete", "build", build.Build.Name, "buildRun", buildRun.Name, "image", spec.Image)
	return res, nil
}

// PatchBuildTriggerFromGitHubEvent looks up the latest push GitHubEvent for
// the given Build's repository and patches the Build's annotations with the
// trigger metadata. No-op if no matching event with payload exists.
//
// This is called by the provider before creating the BuildRun so the
// execution hash includes the correct commit SHA.
func PatchBuildTriggerFromGitHubEvent(ctx context.Context, c client.Client, build *buildv1.Build) error {
	appName := build.Labels["environments.blanketops.dev/name"]
	if appName == "" {
		return nil
	}

	var events eventsv1alpha1.GitHubEventList
	if err := c.List(ctx, &events,
		client.MatchingLabels{"environments.blanketops.dev/name": appName},
	); err != nil {
		return fmt.Errorf("list githubevents: %w", err)
	}

	var latestEvent *eventsv1alpha1.GitHubEvent
	var latestResolved *githubeventResolution.ResolvedGitHubEvent

	for i := range events.Items {
		ev := &events.Items[i]

		resolved, err := githubeventResolution.ResolveGitHubEvent(ev)
		if err != nil {
			continue
		}

		contract := resolved.Spec.ToGitHubEventContract()
		if contract.GetEventId() == "" {
			continue
		}
		if contract.GetEventType().String() != "push" {
			continue
		}

		if latestEvent == nil || ev.CreationTimestamp.After(latestEvent.CreationTimestamp.Time) {
			latestEvent = ev
			latestResolved = resolved
		}
	}

	if latestResolved == nil {
		return nil
	}

	original := build.DeepCopy()
	if build.Annotations == nil {
		build.Annotations = map[string]string{}
	}

	ghContract := latestResolved.Spec.ToGitHubEventContract()
	build.Annotations[triggerTypeAnnotation] = string(ghContract.GetEventType().Type)
	build.Annotations[triggerRefAnnotation] = ghContract.GetRef()
	build.Annotations[triggerSHAAnnotation] = string(ghContract.GetCommitSha())
	build.Annotations[triggerSourceAnnotation] = "github"

	if err := c.Patch(ctx, build, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("patch build trigger annotations: %w", err)
	}

	return nil
}

// Teardown deletes the Shipwright Build and all BuildRuns this provider
// created for the given Build CR. Unlike Run, which creates at most one
// BuildRun per execution hash, Teardown removes every BuildRun labeled with
// this Build's name — CreateBuildRunSpec names runs by hash, so multiple
// runs can accumulate across generations and all must be cleared.
//
// Idempotent — a missing Build or empty BuildRun list is not an error.
// Ownership is via controllerutil.SetControllerReference, so Kubernetes GC
// would eventually reclaim these once the parent Build CR is deleted, but
// teardown deletes explicitly and synchronously rather than depending on
// GC timing to complete before the finalizer is removed.
func (p *KanikoProvider) Teardown(ctx context.Context, build *buildResolution.ResolvedBuild) error {
	if build == nil || build.Build == nil {
		return nil
	}

	name := client.ObjectKeyFromObject(build.Build)

	// ------------------------------------------------
	// Delete all BuildRuns for this Build (may be more than one).
	// ------------------------------------------------
	var runs shipwrightv1alpha1.BuildRunList
	if err := p.Client.List(ctx, &runs,
		client.InNamespace(build.Build.Namespace),
		client.MatchingLabels{"build.blanketops.dev/name": build.Build.Name},
	); err != nil {
		return fmt.Errorf("list buildruns for teardown: %w", err)
	}

	for i := range runs.Items {
		run := &runs.Items[i]
		if err := p.Client.Delete(ctx, run); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete buildrun %s: %w", run.Name, err)
		}
	}

	// ------------------------------------------------
	// Delete the Shipwright Build.
	// ------------------------------------------------
	shipBuild := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      build.Build.Name,
			Namespace: build.Build.Namespace,
		},
	}
	if err := p.Client.Delete(ctx, shipBuild); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete build %s: %w", name.Name, err)
	}

	p.Log.Info("provider.teardown: complete", "build", build.Build.Name)
	return nil
}
