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

package githubevent

import (
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

func TestToGitHubEventContract_NilSpec(t *testing.T) {
	var s *ResolvedGitHubEventSpec
	if got := s.ToGitHubEventContract(); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToGitHubEventContract_Populated(t *testing.T) {
	s := &ResolvedGitHubEventSpec{
		Repository: "x/y",
		EventType:  "push",
		EventID:    "evt-1",
		Ref:        "refs/heads/main",
		CommitSHA:  "abc123",
		Actor:      "neo",
	}
	out := s.ToGitHubEventContract()
	if out.Repository != "x/y" || out.Ref != "refs/heads/main" || out.CommitSha != "abc123" ||
		out.Actor != "neo" || out.EventId != "evt-1" {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.EventType.Type != commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PUSH {
		t.Fatalf("unexpected EventType: %v", out.EventType.Type)
	}
}

func TestNormalizeEventType(t *testing.T) {
	cases := map[string]commoncontractv1.GitHubEventType_GitHubEventType{
		"push":         commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PUSH,
		"pull_request": commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_PULL_REQUEST,
		"release":      commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_RELEASE,
		"manual":       commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_MANUAL,
		"bogus":        commoncontractv1.GitHubEventType_GIT_HUB_EVENT_TYPE_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := normalizeEventType(in); got.Type != want {
			t.Fatalf("normalizeEventType(%q) = %v, want %v", in, got.Type, want)
		}
	}
}
