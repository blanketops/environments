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
(contractv1.GitHubEventSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for event deduplication
  - Event comparison across webhook redeliveries
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.

Event type mapping is non-fatal — unknown GitHub event type strings map to
UNSPECIFIED rather than returning an error. This is intentional: GitHub may
deliver event types the platform has not yet modelled, and dropping them
silently into UNSPECIFIED is preferable to failing the entire resolution
pipeline. New event types should be added to normalizeEventType as they are
onboarded.
*/
package githubevent

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/events/v1alpha1"

// ToGitHubEventContract projects the resolved runtime GitHubEvent spec into a
// protobuf contractv1.GitHubEventSpec for infrastructure consumers
// (hashing, deduplication, comparison, audit).
//
// All fields are projected directly — no transformation beyond event type
// normalisation. CommitSHA and EventID are identity fields that must be
// preserved exactly as received from the GitHub webhook payload.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
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

// normalizeEventType maps a raw GitHub event type string to the corresponding
// contract enum. Unknown event types map to UNSPECIFIED — GitHub may deliver
// event types the platform has not yet modelled and they must not fail
// resolution. Add new cases here as event types are onboarded to the platform.
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
		// Unknown event type — map to UNSPECIFIED rather than failing.
		// GitHub webhook event types are not a closed set from the platform's
		// perspective. Log at the resolution layer if observability is needed.
		return contractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_UNSPECIFIED
	}
}
