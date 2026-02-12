package application

import (
	"context"

	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/buildtrigger"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/domain"
)

type BuildTriggerService struct {
	mapper   *Mapper
	status   *StatusWriter
	provider api.Provider
}

func NewBuildTriggerService(
	mapper *Mapper,
	status *StatusWriter,
	provider api.Provider,
) *BuildTriggerService {
	return &BuildTriggerService{
		mapper:   mapper,
		status:   status,
		provider: provider,
	}
}

func (s *BuildTriggerService) Evaluate(
	ctx context.Context,
	resolved *buildtriggerResolution.ResolvedBuildTrigger,
) error {

	// ------------------------------------------------
	// 1. Map → domain trigger (MUST stay)
	// ------------------------------------------------
	trigger := s.mapper.MapResolvedToDomain(resolved)

	// ------------------------------------------------
	// 2. Delegate evaluation (NO recursion)
	// ------------------------------------------------
	decision, err := s.provider.Evaluate(
		ctx,
		resolved,
		trigger,
	)

	// ------------------------------------------------
	// 3. Write status (against BuildTrigger CR)
	// ------------------------------------------------
	return s.status.Write(
		ctx,
		resolved.Trigger,
		domain.BuildTriggerStatus{
			Accepted: decision.Accepted,
		},
		err,
	)
}
