package githubevent

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/events/v1alpha1"

// ToGitHubEventContract converts a resolved runtime GitHubEvent spec into a
// CONTRACT spec for infra-only / legacy consumers (hashing, comparison, audit).
//
// ⚠️ ONE-WAY adapter.
// Controllers must NEVER consume the returned value.
func (s *ResolvedGitHubEventSpec) ToGitHubEventContract() *contractv1.GitHubEventSpec {
	if s == nil {
		return nil
	}

	return &contractv1.GitHubEventSpec{
		Repository: s.Repository,
		EventType:  normalizeEventType(s.EventType),
		Ref:        s.Ref,
		CommitSha:  s.CommitSHA,
		Actor:      s.Actor,
		EventId:    s.EventID,
	}
}

func normalizeEventType(t string) contractv1.GitHubEventType {
	switch t {
	case "push":
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PUSH
	case "pull_request":
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PULL_REQUEST
	case "release":
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_RELEASE
	case "manual":
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_MANUAL
	default:
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_UNSPECIFIED
	}
}
