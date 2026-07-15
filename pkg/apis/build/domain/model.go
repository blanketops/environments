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
This file owns the core domain types for the build domain — BuildSpec,
BuildPolicy, RetryPolicy, and BuildStatus.

These types are the canonical semantic input and output of build execution.
They are NOT the CRD shape. The Mapper (pkg/build/application/mapper.go)
translates the resolved Build contract into a BuildSpec; the StatusWriter
serialises BuildStatus back into the CR.

All types are fully resolved and immutable at runtime — no defaults are
applied after construction.
*/
package domain

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// BuildSpec is the canonical semantic input to build execution. It is produced
// by the Mapper from a resolved Build contract and consumed by the provider layer.
type BuildSpec struct {
	// Source — the Git repository to build from.
	SourceURL   string
	ContextDir  string
	Revision    string
	CloneSecret string // optional; empty means public or default credentials

	// Strategy — the Shipwright ClusterBuildStrategy to use.
	StrategyName string
	StrategyKind string

	// Output — the fully qualified target image reference.
	Image string

	// ServiceAccount — credentials for pushing the built image.
	// Both fields are optional; empty means Shipwright uses the namespace default.
	ServiceAccountName   string
	ServiceAccountSecret string

	// TimeoutSeconds is optional — nil defers to the provider default.
	TimeoutSeconds *int64

	// Params are strategy-specific parameters passed through to Shipwright.
	Params map[string]string

	// Policy controls retry behaviour for failed executions.
	Policy *BuildPolicy
}

// BuildPolicy is the domain-level execution policy. Fully resolved and
// immutable at runtime — constructed from the resolved Build contract by
// the Mapper.
type BuildPolicy struct {
	Retry *RetryPolicy
}

// RetryPolicy defines retry semantics for failed build executions.
type RetryPolicy struct {
	// OnFailure enables automatic retry when the BuildRun fails.
	OnFailure bool
	// MaxAttempts is the total number of execution attempts including the
	// first run. Must be > 0 when OnFailure is true — enforced at resolution.
	MaxAttempts int32
}

// BuildStatus is the internal domain status for a build execution. It is
// serialised into the Build CR's status.contract field by the StatusWriter
// and read by downstream automation (BuildTrigger, SupplyChain observers).
type BuildStatus struct {
	// Triggered indicates that a BuildRun was dispatched.
	Triggered bool
	// Success indicates that the BuildRun completed successfully.
	// False when Triggered=true — the build has been dispatched but not yet
	// confirmed. The buildrun observer sets Success=true on completion.
	Success bool
	// Message is a human-readable summary of the current state.
	Message string
	// ExecutionRef is the name of the Shipwright BuildRun created for this
	// execution cycle.
	ExecutionRef string
	// BuildHash is the deterministic execution identity computed from the
	// resolved spec and trigger context. Used for BuildRun deduplication.
	BuildHash string
	// OnFailure mirrors the retry policy for observer-layer retry decisions.
	OnFailure bool
	// LastFailureAt is the timestamp of the most recent failure.
	// Optional — populated by the buildrun observer for observability only.
	LastFailureAt *metav1.Time
}
