package domain

import "time"

// Package represents the domain-level truth of a Package.
// It is independent of Kubernetes API machinery.
type PackageSpec struct {
	// ---------------------------------------------------------------------
	// Identity (stable & deterministic)
	// ---------------------------------------------------------------------

	// ID is a stable internal identifier (e.g. namespace/name).
	ID PackageID

	// Name is the human-readable package name.
	Name string

	// Version is the declared package version.
	Version string

	// ---------------------------------------------------------------------
	// Lifecycle & intent
	// ---------------------------------------------------------------------

	// Enabled determines whether reconciliation is active.
	Enabled bool

	// Description provides human context.
	Description string

	// Maintainers represent ownership and accountability.
	Maintainers []Maintainer

	// ---------------------------------------------------------------------
	// Source of truth
	// ---------------------------------------------------------------------

	// Source defines where manifests come from.
	Source PackageSource

	// StateRepo defines where rendered / applied state is tracked.
	StateRepo StateRepository

	// ---------------------------------------------------------------------
	// Reconciliation behavior
	// ---------------------------------------------------------------------

	// DiffEnabled determines whether kapp diff must be executed.
	DiffEnabled bool

	// Strategy defines how manifests are applied (e.g. kustomize).
	Strategy ApplyStrategy

	// ---------------------------------------------------------------------
	// Observability
	// ---------------------------------------------------------------------

	// LastReconciledAt records the last reconciliation attempt.
	LastReconciledAt *time.Time
}

// -----------------------------------------------------------------------------
// Identity
// -----------------------------------------------------------------------------

type PackageID struct {
	Namespace string
	Name      string
}

// -----------------------------------------------------------------------------
// Ownership
// -----------------------------------------------------------------------------

type Maintainer struct {
	Name  string
	Email string
}

// -----------------------------------------------------------------------------
// Source of manifests
// -----------------------------------------------------------------------------

type PackageSource struct {
	// Repository URL containing the package manifests.
	RepositoryURL string

	// CredentialsSecret references auth material (opaque to domain).
	CredentialsSecret string
}

// -----------------------------------------------------------------------------
// State tracking (GitOps anchor)
// -----------------------------------------------------------------------------

type StateRepository struct {
	// URL of the state/manifests repository.
	URL string

	// Ref (branch, tag, or commit).
	Ref Ref

	// CloneSecret used to authenticate.
	CloneSecret string

	// Strategy ImageUpdateAutomation rule(i.e, how to apply our manifests from the repo to our target cluster).
	Strategy string

	// Path is the path of the kustomization file when kustomization strategy = true.
	Path string
}

type Ref struct {

	// Branch is the Git branch to use for the manifests repository.
	Branch string

	// Tag is the Git tag to use for the manifests repository.
	Tag string

	// Commit is the specific Git commit SHA to use for the manifests repository.
	Commit string
}

// PackageStatus is the USER-FACING contract status.
// Stable, backend-agnostic, derived from PackageResult.
type PackageStatus struct {
	// -----------------------------------------------------------------
	// Lifecycle
	// -----------------------------------------------------------------

	Phase   PackagePhase
	Success bool
	Message string

	// -----------------------------------------------------------------
	// Desired state (intent → resolution)
	// -----------------------------------------------------------------

	DesiredRef    string
	DesiredCommit string

	// -----------------------------------------------------------------
	// Execution summary (normalized, truthful)
	// -----------------------------------------------------------------

	// Executed indicates whether a deploy tool (kapp) was invoked.
	Executed bool

	// KappAction represents what kapp actually did.
	// This is promoted intentionally (Diff / Apply / None).
	KappAction KappAction

	// Diff represents the diff outcome observed by kapp.
	Diff DiffState

	// -----------------------------------------------------------------
	// Timing
	// -----------------------------------------------------------------

	StartedAt  time.Time
	FinishedAt time.Time
}

// -----------------------------------------------------------------------------
// Application strategy
// -----------------------------------------------------------------------------

type ApplyStrategy string

const (
	StrategyKustomize ApplyStrategy = "kustomize"
	StrategyPlainYAML ApplyStrategy = "plain"
)
