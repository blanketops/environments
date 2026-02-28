package application

import (
	"strings"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/githubevent/domain"
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
