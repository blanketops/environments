package application

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/gitrepository/domain"
	gitrepoResolution "github.com/ntlaletsi70/blanketops-environments/internal/resolution/gitrepository"
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
func (Mapper) MapResolvedToDomain(
	r *gitrepoResolution.ResolvedGitRepository,
) domain.GitRepository {

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
