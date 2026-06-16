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
This file owns BuildResult — the unified return type from every build provider.

BuildResult is provider-agnostic: Buildah, Kaniko, and Buildpacks all return
the same type. The StatusWriter (pkg/build/application/status.go) consumes it
to write contract status and conditions to the Build CR.

Triggered=true, Success=false is the normal post-dispatch state — the build
has been handed off to Shipwright but has not yet completed. The buildrun
observer writes the terminal outcome asynchronously.
*/
package domain

import (
	shipwrightvbeta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildResult is the unified return value from every build provider.
// All fields are optional — providers populate only what is relevant to
// their execution model.
type BuildResult struct {
	// Success is true only when the BuildRun completed successfully.
	// False when Triggered=true — the build has been dispatched, not confirmed.
	Success bool
	// Triggered is true when a BuildRun was created or already existed for
	// this execution hash.
	Triggered bool
	// Message is a human-readable summary set by the provider or observer.
	Message string
	// Logs captures any provider-level log lines. Optional — most providers
	// rely on Shipwright's own log streaming.
	Logs []string
	// ExecutionRef is the name of the Shipwright BuildRun created for this
	// execution cycle.
	ExecutionRef string
	// BuildHash is the deterministic execution identity. Set by the provider
	// and carried through to BuildStatus for deduplication tracking.
	BuildHash string
	// ArtifactRef is the fully qualified image reference produced by the build,
	// including digest when available. Populated by the buildrun observer after
	// successful completion.
	ArtifactRef string
	// ShipwrightBuild and ShipwrightBuildRun carry the live Shipwright objects
	// for callers that need to inspect them directly. Both are optional.
	ShipwrightBuild    *shipwrightvbeta1.Build
	ShipwrightBuildRun *shipwrightvbeta1.BuildRun
	// OnFailure mirrors the retry policy for observer-layer retry decisions.
	OnFailure bool
	// LastFailureAt is the timestamp of the most recent failure.
	// Populated by the buildrun observer for observability only.
	LastFailureAt *metav1.Time
}
