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

package contract

import (
	"testing"

	"github.com/blanketops/environments/resolution/build/resolve"
)

func TestToBuildContract_NilSpec(t *testing.T) {
	var s *resolve.ResolvedBuildSpec
	if got := ToBuildContract(s); got != nil {
		t.Fatalf("expected nil for nil spec, got %+v", got)
	}
}

func TestToBuildContract_Minimal(t *testing.T) {
	s := &resolve.ResolvedBuildSpec{Image: "foo:latest"}
	out := ToBuildContract(s)
	if out.Image != "foo:latest" {
		t.Fatalf("unexpected image: %q", out.Image)
	}
	if out.Source != nil {
		t.Fatal("expected nil Source when Source.URL is empty")
	}
	if out.Strategy != nil {
		t.Fatal("expected nil Strategy when Strategy.Name is empty")
	}
	if out.ServiceAccount != nil {
		t.Fatal("expected nil ServiceAccount when not set")
	}
	if out.Policy != nil {
		t.Fatal("expected nil Policy when not set")
	}
}

func TestToBuildContract_FullyPopulated(t *testing.T) {
	s := &resolve.ResolvedBuildSpec{
		Image:       "foo:latest",
		Githubevent: "ev-1",
		Source: resolve.ResolvedSource{
			URL:         "git@github.com:x/y",
			Revision:    "main",
			ContextDir:  "/app",
			CloneSecret: "clone-secret",
		},
		Strategy: resolve.ResolvedStrategy{Name: "buildpacks", StrategyKind: "ClusterBuildStrategy"},
		ServiceAccount: &resolve.ResolvedServiceAccount{
			Name:   "sa",
			Secret: "sa-secret",
		},
		Policy: &resolve.ResolvedBuildPolicy{
			Retry: &resolve.ResolvedRetryPolicy{OnFailure: true, MaxAttempts: 3},
		},
	}

	out := ToBuildContract(s)

	if out.GithubEvent != "ev-1" {
		t.Fatalf("unexpected GithubEvent: %q", out.GithubEvent)
	}
	if out.Source == nil || out.Source.Url != "git@github.com:x/y" || out.Source.Revision != "main" {
		t.Fatalf("unexpected Source: %+v", out.Source)
	}
	if out.Strategy == nil || out.Strategy.Name != "buildpacks" {
		t.Fatalf("unexpected Strategy: %+v", out.Strategy)
	}
	if out.ServiceAccount == nil || out.ServiceAccount.Name != "sa" {
		t.Fatalf("unexpected ServiceAccount: %+v", out.ServiceAccount)
	}
	if out.Policy == nil || out.Policy.Retry == nil || !out.Policy.Retry.OnFailure || out.Policy.Retry.MaxAttempts != 3 {
		t.Fatalf("unexpected Policy: %+v", out.Policy)
	}
}

func TestToBuildContract_PolicyWithoutRetry(t *testing.T) {
	s := &resolve.ResolvedBuildSpec{Image: "foo", Policy: &resolve.ResolvedBuildPolicy{}}
	out := ToBuildContract(s)
	if out.Policy != nil {
		t.Fatal("expected nil Policy in the contract when Policy.Retry is nil")
	}
}
