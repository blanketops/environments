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

package gitrepository

import (
	"fmt"

	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/sources/v1alpha1"
)

//
// ==============================
// CONTRACT PROJECTION (ONE-WAY)
// ==============================
//
// Resolved runtime → strict contract types
// For infra / hashing / comparison ONLY.
//
// ⚠️ Controllers must NEVER consume the returned value.
//

func (s *ResolvedGitRepositorySpec) ToGitRepositoryContract() (*contractv1.GitRepositorySpec, error) {
	if s == nil {
		return nil, nil
	}

	// ---------------------------------------------
	// Provider normalization (string → enum VALUE)
	// ---------------------------------------------
	provider, err := normalizeGitProvider(s.Provider)
	if err != nil {
		return nil, err
	}

	out := &contractv1.GitRepositorySpec{
		Provider: provider,
		Repository: &contractv1.GitRepositoryRef{
			Owner: s.Repository.Owner,
			Name:  s.Repository.Name,
		},
	}

	// ---------------------------------------------
	// Webhooks normalization
	// []string → []*GitRepositoryWebhook
	// ---------------------------------------------
	for _, w := range s.Webhooks {
		events, err := normalizeGitEvents(w.Events)
		if err != nil {
			return nil, err
		}

		if len(events) == 0 {
			continue
		}

		out.Webhooks = append(
			out.Webhooks,
			&contractv1.GitRepositoryWebhook{
				Events: events, // []GitEventType (VALUES)
			},
		)
	}

	return out, nil
}

//
// ==============================
// NORMALIZERS (PROTO-ALIGNED)
// ==============================
//

// normalizeGitProvider maps runtime strings to
// blanketops.sources.v1alpha1.GitProvider enum VALUES.
func normalizeGitProvider(p string) (contractv1.GitProvider, error) {
	switch p {
	case "github":
		return contractv1.GitProvider_GIT_PROVIDER_GITHUB, nil
	case "gitlab":
		return contractv1.GitProvider_GIT_PROVIDER_GITLAB, nil
	case "bitbucket":
		return contractv1.GitProvider_GIT_PROVIDER_BITBUCKET, nil
	case "generic", "git":
		return contractv1.GitProvider_GIT_PROVIDER_GENERIC_GIT, nil
	default:
		return contractv1.GitProvider_GIT_PROVIDER_UNSPECIFIED,
			fmt.Errorf("unsupported git provider %q", p)
	}
}

// normalizeGitEvents maps runtime event strings to
// blanketops.sources.v1alpha1.GitEventType enum VALUES.
func normalizeGitEvents(events []string) ([]contractv1.GitEventType, error) {

	var out []contractv1.GitEventType

	for _, e := range events {
		switch e {
		case "push":
			out = append(out, contractv1.GitEventType_GIT_EVENT_TYPE_PUSH)

		case "pull_request":
			out = append(out, contractv1.GitEventType_GIT_EVENT_TYPE_PULL_REQUEST)

		case "release":
			out = append(out, contractv1.GitEventType_GIT_EVENT_TYPE_RELEASE)

		case "tag":
			out = append(out, contractv1.GitEventType_GIT_EVENT_TYPE_TAG)

		default:
			return nil, fmt.Errorf("unsupported git event %q", e)
		}
	}

	return out, nil
}
