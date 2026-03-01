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

type DiffState string

const (
	DiffUnknown  DiffState = "Unknown"
	DiffClean    DiffState = "Clean"
	DiffDetected DiffState = "Detected"
	DiffApplied  DiffState = "Applied"
)

type ApplicationPhase string

const (
	ApplicationPhasePending ApplicationPhase = "Pending"
	ApplicationPhaseReady   ApplicationPhase = "Ready"
	ApplicationPhaseFailed  ApplicationPhase = "Failed"
)

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
