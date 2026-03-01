package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
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
