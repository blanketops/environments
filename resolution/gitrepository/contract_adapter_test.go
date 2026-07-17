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
	"strings"
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

func TestToGitRepositoryContract_NilSpec(t *testing.T) {
	var s *ResolvedGitRepositorySpec
	out, err := s.ToGitRepositoryContract()
	if err != nil {
		t.Fatalf("expected no error for nil spec, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil, got %+v", out)
	}
}

func TestToGitRepositoryContract_UnknownProvider(t *testing.T) {
	s := &ResolvedGitRepositorySpec{Provider: "bogus", Repository: GitRepositoryRef{Owner: "o", Name: "n"}}
	_, err := s.ToGitRepositoryContract()
	if err == nil || !strings.Contains(err.Error(), "unsupported git provider") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestToGitRepositoryContract_Minimal(t *testing.T) {
	s := &ResolvedGitRepositorySpec{Provider: "github", Repository: GitRepositoryRef{Owner: "o", Name: "n"}}
	out, err := s.ToGitRepositoryContract()
	if err != nil {
		t.Fatalf("ToGitRepositoryContract: %v", err)
	}
	if out.Provider.Provider != commoncontractv1.GitProvider_GIT_PROVIDER_GITHUB {
		t.Fatalf("unexpected provider: %v", out.Provider.Provider)
	}
	if out.Repository.Owner != "o" || out.Repository.Name != "n" {
		t.Fatalf("unexpected repository: %+v", out.Repository)
	}
	if len(out.Webhooks) != 0 {
		t.Fatalf("expected no webhooks, got %+v", out.Webhooks)
	}
}

func TestToGitRepositoryContract_WebhooksProjected(t *testing.T) {
	s := &ResolvedGitRepositorySpec{
		Provider:   "github",
		Repository: GitRepositoryRef{Owner: "o", Name: "n"},
		Webhooks: []GitRepositoryWebhook{
			{Events: []string{"push", "pull_request", "release", "tag"}},
		},
	}
	out, err := s.ToGitRepositoryContract()
	if err != nil {
		t.Fatalf("ToGitRepositoryContract: %v", err)
	}
	if len(out.Webhooks) != 1 || len(out.Webhooks[0].Events) != 4 {
		t.Fatalf("unexpected webhooks: %+v", out.Webhooks)
	}
}

func TestToGitRepositoryContract_WebhookEmptyEventsSkipped(t *testing.T) {
	s := &ResolvedGitRepositorySpec{
		Provider:   "github",
		Repository: GitRepositoryRef{Owner: "o", Name: "n"},
		Webhooks:   []GitRepositoryWebhook{{Events: []string{}}},
	}
	out, err := s.ToGitRepositoryContract()
	if err != nil {
		t.Fatalf("ToGitRepositoryContract: %v", err)
	}
	if len(out.Webhooks) != 0 {
		t.Fatalf("expected empty-events webhook to be skipped, got %+v", out.Webhooks)
	}
}

func TestToGitRepositoryContract_WebhookUnsupportedEvent(t *testing.T) {
	s := &ResolvedGitRepositorySpec{
		Provider:   "github",
		Repository: GitRepositoryRef{Owner: "o", Name: "n"},
		Webhooks:   []GitRepositoryWebhook{{Events: []string{"bogus"}}},
	}
	_, err := s.ToGitRepositoryContract()
	if err == nil || !strings.Contains(err.Error(), "unsupported git event") {
		t.Fatalf("expected unsupported event error, got %v", err)
	}
}

func TestNormalizeGitProvider_Aliases(t *testing.T) {
	for _, alias := range []string{"generic", "git"} {
		got, err := normalizeGitProvider(alias)
		if err != nil {
			t.Fatalf("normalizeGitProvider(%q): %v", alias, err)
		}
		if got.Provider != commoncontractv1.GitProvider_GIT_PROVIDER_GENERIC_GIT {
			t.Fatalf("expected GENERIC_GIT for %q, got %v", alias, got.Provider)
		}
	}
}

func TestNormalizeGitProvider_All(t *testing.T) {
	cases := map[string]commoncontractv1.GitProvider_GitProvider{
		"github":    commoncontractv1.GitProvider_GIT_PROVIDER_GITHUB,
		"gitlab":    commoncontractv1.GitProvider_GIT_PROVIDER_GITLAB,
		"bitbucket": commoncontractv1.GitProvider_GIT_PROVIDER_BITBUCKET,
	}
	for in, want := range cases {
		got, err := normalizeGitProvider(in)
		if err != nil {
			t.Fatalf("normalizeGitProvider(%q): %v", in, err)
		}
		if got.Provider != want {
			t.Fatalf("normalizeGitProvider(%q) = %v, want %v", in, got.Provider, want)
		}
	}
}
