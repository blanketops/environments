package domain

import "time"

// PackageResult represents the outcome of a single reconciliation run.
type PackageResult struct {
	// ---------------------------------------------------------------------
	// Outcome
	// ---------------------------------------------------------------------

	Success bool
	Phase   PackagePhase
	Message string

	// ---------------------------------------------------------------------
	// Source-of-truth resolution
	// ---------------------------------------------------------------------

	// DesiredRef is what the controller intended to apply.
	DesiredRef string

	// DesiredCommit is the resolved commit hash.
	DesiredCommit string

	// ---------------------------------------------------------------------
	// kapp execution details
	// ---------------------------------------------------------------------

	Kapp KappResult

	// ---------------------------------------------------------------------
	// Timing
	// ---------------------------------------------------------------------

	StartedAt  time.Time
	FinishedAt time.Time
}

// -----------------------------------------------------------------------------
// kapp execution result
// -----------------------------------------------------------------------------

type KappResult struct {
	// Executed indicates whether kapp was invoked at all.
	Name      string
	Namespace string
	Executed  bool

	// Deploy timing (authoritative)
	DeployStartedAt *time.Time
	DeployUpdatedAt *time.Time

	// Exit info
	DeployFinished bool
	DeployExitCode *int

	// Action taken by kapp.
	// e.g. diff, apply, noop
	Action KappAction

	// Diff indicates whether differences were detected.
	Diff DiffState

	// Output is raw kapp stdout/stderr (trimmed upstream if needed).
	Output string

	// Error captures execution failure, if any.
	Error string
}

// -----------------------------------------------------------------------------
// kapp actions
// -----------------------------------------------------------------------------

type KappAction string

const (
	KappActionNone  KappAction = "None"
	KappActionDiff  KappAction = "Diff"
	KappActionApply KappAction = "Apply"
)
