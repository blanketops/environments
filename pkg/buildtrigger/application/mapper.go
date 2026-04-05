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

package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
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
