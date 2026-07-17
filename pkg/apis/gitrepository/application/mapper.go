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
	"fmt"

	"github.com/blanketops/environments/pkg/apis/gitrepository/domain"
	gitrepoResolution "github.com/blanketops/environments/resolution/gitrepository/resolve"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

// MapResolvedToDomain converts a fully resolved GitRepository into
// a pure domain GitRepository.
//
// CONTRACT:
// - Resolver guarantees presence of mandatory fields
// - Nil / empty mandatory values indicate a resolver bug and MUST panic
// - Optional fields must be preserved verbatim
// - Mapper must not invent defaults or reinterpret intent
func (Mapper) MapResolvedToDomain(r *gitrepoResolution.ResolvedGitRepository) domain.GitRepository {

	spec := r.Spec

	// ---------------------------------------------------------------------
	// INVARIANTS (resolver-owned guarantees)
	// ---------------------------------------------------------------------
	if spec.Provider == "" {
		panic(fmt.Sprintf(
			"resolved gitrepository %q has empty Provider (resolver bug)",
			r.Repository.Name,
		))
	}

	if spec.Repository.Owner == "" {
		panic(fmt.Sprintf(
			"resolved gitrepository %q has empty Repository.Owner (resolver bug)",
			r.Repository.Name,
		))
	}

	if spec.Repository.Name == "" {
		panic(fmt.Sprintf(
			"resolved gitrepository %q has empty Repository.Name (resolver bug)",
			r.Repository.Name,
		))
	}

	// ---------------------------------------------------------------------
	// Webhooks (OPTIONAL – preserved exactly)
	// ---------------------------------------------------------------------
	webhooks := make([]domain.WebhookSpec, 0, len(spec.Webhooks))

	for _, wh := range spec.Webhooks {
		events := make([]domain.EventType, 0, len(wh.Events))
		for _, e := range wh.Events {
			events = append(events, domain.EventType(e))
		}

		webhooks = append(webhooks, domain.WebhookSpec{
			Events: events,
		})
	}

	// ---------------------------------------------------------------------
	// Domain boundary mapping
	// ---------------------------------------------------------------------
	return domain.GitRepository{
		Provider: domain.Provider(spec.Provider),

		Repository: domain.RepositoryID{
			Owner: spec.Repository.Owner,
			Name:  spec.Repository.Name,
		},

		Webhooks: webhooks,
	}
}
