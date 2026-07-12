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
This file owns the Mapper for the GitHubEvent domain — the pure translation
layer between the resolved GitHubEvent contract and the domain.GitHubEvent type
consumed by the application service and provider.

The Mapper is copy-only: no defaults, no inference, no validation. Repository
is split from "owner/name" into Owner and Name as a normalisation step — if the
format is invalid, Owner is empty and Name carries the raw string.

Panics on nil input — a nil ResolvedGitHubEvent indicates a resolver bug.
*/
package application

import (
	"strings"

	"github.com/BlanketOps/environments/pkg/apis/githubevent/domain"
	githubeventResolution "github.com/BlanketOps/environments/resolution/githubevent"
)

// Mapper converts a ResolvedGitHubEvent into a pure domain.GitHubEvent.
// Stateless and safe for concurrent use.
type Mapper struct{}

// NewMapper constructs a Mapper.
func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain translates a ResolvedGitHubEvent into a domain.GitHubEvent.
// Panics if resolved or resolved.Spec is nil — these indicate a resolver bug.
func (Mapper) MapResolvedToDomain(r *githubeventResolution.ResolvedGitHubEvent) domain.GitHubEvent {
	if r == nil || r.Spec == nil {
		panic("nil ResolvedGitHubEvent (resolver bug)")
	}

	owner, name := splitRepo(r.Spec.Repository)

	return domain.GitHubEvent{
		Type: domain.EventType(r.Spec.EventType),
		Repository: domain.Repository{
			FullName: r.Spec.Repository,
			Owner:    owner,
			Name:     name,
		},
		Ref:    domain.GitRef{Name: r.Spec.Ref},
		Commit: domain.Commit{SHA: r.Spec.CommitSHA},
		Actor:  domain.Actor{Login: r.Spec.Actor},
	}
}

// splitRepo splits "owner/name" into its components. Normalisation only —
// validation is the resolver's responsibility. Returns empty owner when the
// format is unexpected.
func splitRepo(full string) (owner, name string) {
	parts := strings.SplitN(full, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", full
}
