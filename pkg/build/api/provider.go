package api

import (
	"context"

	buildResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/build"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/build/domain"
)

type Provider interface {
	// Run executes a build using a resolved build and pure domain spec.
	//
	// - resolved.Build → metadata / owner refs / namespace
	// - resolved.Spec  → typed contract semantics
	// - spec           → internal domain projection
	Run(
		ctx context.Context,
		resolved *buildResolution.ResolvedBuild,
		spec domain.BuildSpec,
	) (domain.BuildResult, error)
}
