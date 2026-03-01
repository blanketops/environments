package application

import (
	"context"

	bldResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
)

type BuildService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

func NewBuildService(
	mapper *Mapper,
	status *StatusWriter,
	backend *BackendSelector,
) *BuildService {
	return &BuildService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

func (s *BuildService) Reconcile(
	ctx context.Context,
	resolved *bldResolution.ResolvedBuild,
) error {

	// ------------------------------------------------
	// 1. Map → domain spec
	// ------------------------------------------------
	spec := s.mapper.MapResolvedToDomain(resolved)

	// ------------------------------------------------
	// 2. Select backend
	// ------------------------------------------------
	provider := s.backend.ForSpec(spec)

	// ------------------------------------------------
	// 3. Execute build
	// ------------------------------------------------
	result, err := provider.Run(ctx, resolved, spec)

	// ------------------------------------------------
	// 4. Write status (still against CR)
	// ------------------------------------------------
	return s.status.Write(ctx, resolved.Build, result, err)
}
