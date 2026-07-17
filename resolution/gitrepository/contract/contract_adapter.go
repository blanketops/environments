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
Package contract implements the GitRepository CR contract adapter — the
one-way projection from the resolved runtime spec
(resolve.ResolvedGitRepositorySpec) to the protobuf contract type
(sourcescontractv1alpha1.GitRepositorySpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for repository deduplication
  - Spec comparison across reconciliation cycles
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.

The common proto wrapper pattern applies throughout:
  - Provider → *v1.GitProvider{Provider: enumVal}
  - Webhook events → []*v1.GitEventType{{Type: enumVal}}

GitRepositoryRef and GitRepositoryWebhook are local messages in the
sources v1alpha1 package, not common types.

Provider mapping is fatal on unknown values — providers are a
platform-controlled closed set. Event type mapping is also fatal —
unknown event strings indicate a misconfigured webhook spec.
*/
package contract

import (
	"fmt"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	sourcescontractv1alpha1 "github.com/blanketops/environments-contract/blanketops/sources/v1alpha1"

	"github.com/blanketops/environments/resolution/gitrepository/resolve"
)

// ToGitRepositoryContract projects the resolved runtime GitRepository spec
// into a protobuf sourcescontractv1alpha1.GitRepositorySpec for infrastructure
// consumers (hashing, comparison, audit).
//
// Provider and webhook event types are projected as wrapper message instances.
// Returns an error when the provider string is unknown or any webhook event
// string is unsupported.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func ToGitRepositoryContract(s *resolve.ResolvedGitRepositorySpec) (*sourcescontractv1alpha1.GitRepositorySpec, error) {
	if s == nil {
		return nil, nil
	}

	// Provider is required — unknown values fail the projection.
	provider, err := normalizeGitProvider(s.Provider)
	if err != nil {
		return nil, err
	}

	out := &sourcescontractv1alpha1.GitRepositorySpec{
		Provider: provider,
		Repository: &sourcescontractv1alpha1.GitRepositoryRef{
			Owner: s.Repository.Owner,
			Name:  s.Repository.Name,
		},
	}

	// ------------------------------------------------
	// Webhook projection.
	//
	// Each event string maps to a *v1.GitEventType wrapper message.
	// Webhooks that produce zero valid events are skipped — an empty
	// events list is not a valid webhook configuration.
	// ------------------------------------------------
	for _, w := range s.Webhooks {
		events, err := normalizeGitEvents(w.Events)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			continue
		}
		out.Webhooks = append(out.Webhooks, &sourcescontractv1alpha1.GitRepositoryWebhook{
			Events: events,
		})
	}

	return out, nil
}

// -----------------------------------------------------------------------------
// Normalizers: runtime string → proto wrapper message
// -----------------------------------------------------------------------------

// normalizeGitProvider maps a runtime provider string to a
// *v1.GitProvider wrapper message. Returns an error for unknown values —
// providers are a platform-controlled closed set.
//
// "generic" and "git" are aliases for GIT_PROVIDER_GENERIC_GIT.
func normalizeGitProvider(p string) (*commoncontractv1.GitProvider, error) {
	switch p {
	case "github":
		return &commoncontractv1.GitProvider{
			Provider: commoncontractv1.GitProvider_GIT_PROVIDER_GITHUB,
		}, nil
	case "gitlab":
		return &commoncontractv1.GitProvider{
			Provider: commoncontractv1.GitProvider_GIT_PROVIDER_GITLAB,
		}, nil
	case "bitbucket":
		return &commoncontractv1.GitProvider{
			Provider: commoncontractv1.GitProvider_GIT_PROVIDER_BITBUCKET,
		}, nil
	case "generic", "git":
		return &commoncontractv1.GitProvider{
			Provider: commoncontractv1.GitProvider_GIT_PROVIDER_GENERIC_GIT,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported git provider %q", p)
	}
}

// normalizeGitEvents maps a slice of runtime event strings to
// []*v1.GitEventType wrapper message instances. Returns an error on the
// first unsupported string — event types are a platform-controlled closed set.
func normalizeGitEvents(events []string) ([]*commoncontractv1.GitEventType, error) {
	var out []*commoncontractv1.GitEventType
	for _, e := range events {
		switch e {
		case "push":
			out = append(out, &commoncontractv1.GitEventType{
				Type: commoncontractv1.GitEventType_GIT_EVENT_TYPE_PUSH,
			})
		case "pull_request":
			out = append(out, &commoncontractv1.GitEventType{
				Type: commoncontractv1.GitEventType_GIT_EVENT_TYPE_PULL_REQUEST,
			})
		case "release":
			out = append(out, &commoncontractv1.GitEventType{
				Type: commoncontractv1.GitEventType_GIT_EVENT_TYPE_RELEASE,
			})
		case "tag":
			out = append(out, &commoncontractv1.GitEventType{
				Type: commoncontractv1.GitEventType_GIT_EVENT_TYPE_TAG,
			})
		default:
			return nil, fmt.Errorf("unsupported git event %q", e)
		}
	}
	return out, nil
}
