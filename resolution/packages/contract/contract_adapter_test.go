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

	"github.com/blanketops/environments/resolution/packages/resolve"
)

func TestToPackageContract_NilSpec(t *testing.T) {
	var s *resolve.ResolvedPackageSpec
	if got := ToPackageContract(s); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToPackageContract_Minimal(t *testing.T) {
	s := &resolve.ResolvedPackageSpec{
		Name:              "pkg1",
		Version:           "1.0.0",
		PackageRepository: resolve.ResolvedPackageRepository{URL: "oci://x"},
	}
	out := ToPackageContract(s)
	if out.Name != "pkg1" || out.Repository.Url != "oci://x" {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.StateRepository != nil {
		t.Fatal("expected nil StateRepository when not set")
	}
	if len(out.Maintainers) != 0 {
		t.Fatalf("expected no maintainers, got %+v", out.Maintainers)
	}
}

func TestToPackageContract_FullyPopulated(t *testing.T) {
	s := &resolve.ResolvedPackageSpec{
		Enabled:           true,
		Name:              "pkg1",
		Version:           "1.0.0",
		Description:       "a package",
		DiffEnabled:       true,
		PackageRepository: resolve.ResolvedPackageRepository{URL: "oci://x", CredentialsSecret: "sec"},
		StateRepository: &resolve.ResolvedStateRepository{
			URL: "git@x", CloneSecret: "clone-sec", Strategy: "kapp", Path: "/state",
		},
		Maintainers: []resolve.ResolvedMaintainer{{Name: "neo", Email: "neo@x.com"}},
	}
	out := ToPackageContract(s)
	if out.Description != "a package" || !out.DiffEnabled {
		t.Fatalf("unexpected contract: %+v", out)
	}
	if out.StateRepository == nil || out.StateRepository.Url != "git@x" || out.StateRepository.Path != "/state" {
		t.Fatalf("unexpected StateRepository: %+v", out.StateRepository)
	}
	if len(out.Maintainers) != 1 || out.Maintainers[0].Name != "neo" || out.Maintainers[0].Email != "neo@x.com" {
		t.Fatalf("unexpected Maintainers: %+v", out.Maintainers)
	}
}
