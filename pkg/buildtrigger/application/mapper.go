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

/*
This file owns the Mapper for the BuildTrigger domain — the pure translation
layer between the resolved BuildTrigger contract and the domain.BuildTrigger
type consumed by the application service and provider layer.

The Mapper is copy-only: no time.Now(), no ID generation, no inference. All
fields originate from the resolved spec. If the resolver provides it, the
Mapper copies it; if the resolver omits it, it is absent in the domain model.

Panics on nil input — a nil ResolvedBuildTrigger indicates a resolver bug and
must surface loudly rather than silently producing an empty domain model.
*/
package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	buildtriggerResolution "github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
)

// Mapper converts a ResolvedBuildTrigger into a pure domain.BuildTrigger.
// It is stateless and safe for concurrent use.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain translates a ResolvedBuildTrigger into a domain.BuildTrigger.
// Panics if resolved or resolved.Spec is nil — these indicate a resolver bug.
func (Mapper) MapResolvedToDomain(
	resolved *buildtriggerResolution.ResolvedBuildTrigger,
) domain.BuildTrigger {
	if resolved == nil || resolved.Spec == nil {
		panic("nil ResolvedBuildTrigger (resolver bug)")
	}

	trigger := domain.Trigger{
		ID:          resolved.Spec.TriggerID,
		Source:      resolved.Spec.Source,
		Type:        resolved.Spec.Type,
		Repository:  resolved.Spec.Repository,
		Ref:         resolved.Spec.Ref,
		SHA:         resolved.Spec.SHA,
		Actor:       resolved.Spec.Actor,
		EventID:     resolved.Spec.EventID,
		PayloadHash: resolved.Spec.PayloadHash,
		OccurredAt:  resolved.Spec.OccurredAt,
		ReceivedAt:  resolved.Spec.ReceivedAt,
	}

	// Target.Name is the buildRef — the Build CR this trigger will fire.
	// Resolved in the same namespace as the BuildTrigger CR.
	target := domain.Target{
		Kind:      domain.TargetKindBuild,
		Name:      resolved.Spec.Target.Name,
		Namespace: resolved.Spec.Target.Namespace,
	}

	return domain.BuildTrigger{
		Trigger: trigger,
		Target:  target,
	}
}
