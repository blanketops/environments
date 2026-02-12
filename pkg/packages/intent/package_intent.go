package intent

import (
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/domain"
)

// PackageIntent is the compiled, immutable execution plan.
type PackageIntent struct {
	// Stable identity
	ID domain.PackageID

	// Source of manifests
	Source domain.PackageSource

	// State repository (GitOps anchor)
	StateRepo domain.StateRepository

	// Execution behavior
	DiffEnabled bool
	Strategy    domain.ApplyStrategy

	// Resolved ref (branch/tag/commit)
	ResolvedRef    string
	ResolvedCommit string
}
