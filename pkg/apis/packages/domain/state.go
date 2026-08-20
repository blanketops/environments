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

package domain

import "time"

// PackageState represents the observed state of a Package.
type PackageState struct {
	// ---------------------------------------------------------------------
	// High-level phase
	// ---------------------------------------------------------------------

	Phase PackagePhase

	// Message provides human-readable context.
	Message string

	// ---------------------------------------------------------------------
	// Git / source-of-truth tracking
	// ---------------------------------------------------------------------

	// AppliedRef is the git ref (branch/tag/commit) last applied.
	AppliedRef string

	// AppliedCommit is the exact commit hash applied.
	AppliedCommit string

	// ---------------------------------------------------------------------
	// kapp-specific observability
	// ---------------------------------------------------------------------

	// Diff indicates whether kapp detected differences.
	Diff DiffState

	// LastDiff contains the most recent kapp diff output (if enabled).
	LastDiff string

	// ---------------------------------------------------------------------
	// Timestamps
	// ---------------------------------------------------------------------

	LastAppliedAt *time.Time
	LastCheckedAt *time.Time
}

// -----------------------------------------------------------------------------
// Phase
// -----------------------------------------------------------------------------

// PackagePhase is the high-level reconciliation phase of a Package.
type PackagePhase string

const (
	PackagePhasePending     PackagePhase = "Pending"
	PackagePhaseApplied     PackagePhase = "Applied"
	PackagePhaseDrifted     PackagePhase = "Drifted"
	PackagePhaseFailed      PackagePhase = "Failed"
	PackagePhaseDisabled    PackagePhase = "Disabled"
	PackagePhaseReconciling PackagePhase = "Reconciling"
	PackagePhaseReady       PackagePhase = "Ready"
	PackagePhaseSucceeded   PackagePhase = "Succeeded"
	PackagePhaseUnknown     PackagePhase = "Unknown"
)

// -----------------------------------------------------------------------------
// Diff state
// -----------------------------------------------------------------------------

// DiffState is the outcome of the most recent kapp diff check.
type DiffState string

const (
	DiffUnknown  DiffState = "Unknown"
	DiffClean    DiffState = "Clean"
	DiffDetected DiffState = "Detected"
	DiffApplied  DiffState = "Applied"
)

// ApplicationPhase is the observed lifecycle phase of the kapp-controller
// App CR backing a Package.
type ApplicationPhase string

const (
	ApplicationPhasePending ApplicationPhase = "Pending"
	ApplicationPhaseReady   ApplicationPhase = "Ready"
	ApplicationPhaseFailed  ApplicationPhase = "Failed"
)

// ApplicationState is the state observed from the kapp-controller App CR
// backing a Package, before it is mapped onto PackageResult/PackageStatus.
type ApplicationState struct {
	Name      string
	Namespace string

	Phase   ApplicationPhase
	Message string

	// Deploy timing (authoritative)
	DeployStartedAt *time.Time
	DeployUpdatedAt *time.Time

	// Exit info
	DeployFinished bool
	DeployExitCode *int
}
