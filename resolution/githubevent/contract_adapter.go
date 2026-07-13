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
Package githubevent implements resolution for the GitHubEvent CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedGitHubEventSpec) to the protobuf contract type
(eventscontractv1alpha1.GitHubEventSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for event deduplication
  - Event comparison across webhook redeliveries
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.

Event type mapping is non-fatal — unknown GitHub event type strings map to
UNSPECIFIED rather than returning an error. GitHub may deliver event types
the platform has not yet modelled; dropping them to UNSPECIFIED is preferable
to failing the entire resolution pipeline.

The common proto wrapper pattern applies: EventType is projected as
*v1.GitHubEventType with the enum value on the Type field.
*/
package githubevent

import (
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	eventscontractv1alpha1 "github.com/blanketops/environments-contract/blanketops/events/v1alpha1"
)

// ToGitHubEventContract projects the resolved runtime GitHubEvent spec into a
// protobuf eventscontractv1alpha1.GitHubEventSpec for infrastructure consumers
// (hashing, deduplication, comparison, audit).
//
// EventType is projected as a *v1.GitHubEventType wrapper message with the
// enum value on the Type field.
//
// CommitSha and EventId are identity fields preserved exactly as received
// from the GitHub webhook payload — no transformation applied.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedGitHubEventSpec) ToGitHubEventContract() *eventscontractv1alpha1.GitHubEventSpec {
	if s == nil {
		return nil
	}
	// out := &eventscontractv1alpha1.GitHubEventSpec{}
	// // ------------------------------------------------
	// // Repository (Recursive null check)
	// // ------------------------------------------------
	// if s.Repository.FullName != "" {
	// 	out.Repository = &eventscontractv1alpha1.Repository{
	// 		Owner:    s.Repository.Owner,
	// 		Name:     s.Repository.Name,
	// 		FullName: s.Repository.FullName,
	// 	}
	// }

	// // ------------------------------------------------
	// // Event Type
	// // ------------------------------------------------
	// if s.EventType != "" {
	// 	out.EventType = normalizeEventType(s.EventType)
	// }

	// // ------------------------------------------------
	// // Git Reference
	// // ------------------------------------------------
	// if s.Ref.Name != "" {
	// 	out.Ref = &eventscontractv1alpha1.GitRef{
	// 		Name: s.Ref.Name,
	// 	}
	// }

	// // ------------------------------------------------
	// // Identity Fields (Commit, Actor, EventID)
	// // ------------------------------------------------
	// if s.CommitSHA != "" {
	// 	out.CommitSha = s.CommitSHA
	// }

	// if s.Actor != "" {
	// 	out.Actor = s.Actor
	// }

	// if s.EventID != "" {
	// 	out.EventId = s.EventID
	// }

	// return out
	return &eventscontractv1alpha1.GitHubEventSpec{
		Repository: s.Repository,
		EventType:  normalizeEventType(s.EventType),
		Ref:        s.Ref,
		CommitSha:  s.CommitSHA,
		Actor:      s.Actor,
		EventId:    s.EventID,
	}
}

// normalizeEventType maps a raw GitHub event type string to the corresponding
// *v1.GitHubEventType wrapper message. Unknown event types map to UNSPECIFIED
// — GitHub webhook event types are not a closed set from the platform's
// perspective. Add new cases here as event types are onboarded.
func normalizeEventType(t string) *commoncontractv1.GitHubEventType {
	switch t {
	case "push":
		return &commoncontractv1.GitHubEventType{
			Type: commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PUSH,
		}
	case "pull_request":
		return &commoncontractv1.GitHubEventType{
			Type: commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PULL_REQUEST,
		}
	case "release":
		return &commoncontractv1.GitHubEventType{
			Type: commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_RELEASE,
		}
	case "manual":
		return &commoncontractv1.GitHubEventType{
			Type: commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_MANUAL,
		}
	default:
		// Unknown event type — map to UNSPECIFIED rather than failing.
		// Log at the resolution layer if observability is needed.
		return &commoncontractv1.GitHubEventType{
			Type: commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_UNSPECIFIED,
		}
	}
}
