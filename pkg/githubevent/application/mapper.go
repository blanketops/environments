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
	"strings"

	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// Mapper converts resolved GitHubEvents into pure domain models.
type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain maps a ResolvedGitHubEvent into a domain.GitHubEvent.
//
// CONTRACT:
// - Input is fully resolved and authoritative
// - Mapper must not read from CR spec fields
// - No defaults, no inference, no validation
func (Mapper) MapResolvedToDomain(
	r *githubeventResolution.ResolvedGitHubEvent,
) domain.GitHubEvent {
	if r == nil || r.Spec == nil {
		panic("nil ResolvedGitHubEvent (resolver bug)")
	}

	owner, name := splitRepo(r.Spec.Repository)

	return domain.GitHubEvent{
		//EventID: domain.EventID,
		Type: domain.EventType(r.Spec.EventType),

		Repository: domain.Repository{
			FullName: r.Spec.Repository,
			Owner:    owner,
			Name:     name,
		},

		Ref: domain.GitRef{
			Name: r.Spec.Ref,
		},

		Commit: domain.Commit{
			SHA: r.Spec.CommitSHA,
		},

		Actor: domain.Actor{
			Login: r.Spec.Actor,
		},
	}
}

// splitRepo splits "owner/name" into owner and name.
// Normalization only — validation belongs elsewhere.
func splitRepo(full string) (owner, name string) {
	parts := strings.SplitN(full, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", full
}
