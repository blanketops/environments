package application

import (
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/buildtrigger"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/buildtrigger/domain"
)

// Mapper converts resolved BuildTriggers into pure domain models.
// PURE. Side-effect free. Copy-only.
type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain maps a ResolvedBuildTrigger into a domain.BuildTrigger.
//
// CONTRACT:
// - Input is fully resolved and authoritative
// - No time.Now()
// - No ID generation
// - No inference
func (Mapper) MapResolvedToDomain(
	resolved *buildtriggerResolution.ResolvedBuildTrigger,
) domain.BuildTrigger {

	if resolved == nil || resolved.Spec == nil {
		panic("nil ResolvedBuildTrigger (resolver bug)")
	}

	trigger := domain.Trigger{
		ID: resolved.Spec.TriggerID,

		Source: resolved.Spec.Source,
		Type:   resolved.Spec.Type,

		Repository: resolved.Spec.Repository,
		Ref:        resolved.Spec.Ref,
		SHA:        resolved.Spec.SHA,
		Actor:      resolved.Spec.Actor,
		EventID:    resolved.Spec.EventID,

		PayloadHash: resolved.Spec.PayloadHash,

		OccurredAt: resolved.Spec.OccurredAt,
		ReceivedAt: resolved.Spec.ReceivedAt,
	}

	target := domain.Target{
		Kind:      domain.TargetKindBuild,
		Name:      resolved.Spec.Target.Name, // ← this IS the buildref
		Namespace: resolved.Spec.Target.Namespace,
	}

	return domain.BuildTrigger{
		Trigger: trigger,
		Target:  target,
	}
}
