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

package resolve

import (
	"strings"
	"testing"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func pkgWithContract(raw string) *environmentv1alpha1.Package {
	return &environmentv1alpha1.Package{
		Spec: environmentv1alpha1.PackageSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"name":"pkg1","version":"1.0.0","repository":{"url":"oci://x"}}`

func TestResolvePackage_Nil(t *testing.T) {
	if _, err := ResolvePackage(nil); err == nil {
		t.Fatal("expected error for nil package")
	}
}

func TestResolvePackage_EmptyContract(t *testing.T) {
	p := &environmentv1alpha1.Package{}
	if _, err := ResolvePackage(p); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolvePackage_InvalidJSON(t *testing.T) {
	p := pkgWithContract(`{not json`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolvePackage_NameMissing(t *testing.T) {
	p := pkgWithContract(`{"version":"1.0.0","repository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected packageName error, got %v", err)
	}
}

func TestResolvePackage_VersionMissing(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","repository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected packageVersion error, got %v", err)
	}
}

func TestResolvePackage_RepositoryMissing(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("expected packageRepository error, got %v", err)
	}
}

func TestResolvePackage_RepositoryWrongType(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":"not-an-object"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), `field "repository" must be an object`) {
		t.Fatalf("expected packageRepository type error, got %v", err)
	}
}

func TestResolvePackage_RepositoryURLMissing(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Fatalf("expected url error, got %v", err)
	}
}

func TestResolvePackage_MinimalValid(t *testing.T) {
	p := pkgWithContract(minimalValid)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	if resolved.Spec.Name != "pkg1" || resolved.Spec.Version != "1.0.0" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
	if resolved.Spec.DiffEnabled {
		t.Fatal("expected DiffEnabled to default to false")
	}
	if resolved.Spec.StateRepository != nil {
		t.Fatal("expected nil StateRepository when not declared")
	}
	if resolved.Spec.Maintainers != nil {
		t.Fatal("expected nil Maintainers when not declared")
	}
}

// TestResolvePackage_NoEnabledSwitch covers a contract that still carries an
// "enabled" key. There is no such switch: a Package that exists is
// reconciled, and removing it is how it is stopped. The key is ignored.
func TestResolvePackage_NoEnabledSwitch(t *testing.T) {
	p := pkgWithContract(`{"enabled":false,"name":"pkg1","version":"1.0.0","repository":{"url":"x"}}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	if resolved.Spec.Name != "pkg1" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
}

func TestResolvePackage_StateRepositoryWrongType(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"stateRepository":"not-an-object"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "stateRepository must be an object") {
		t.Fatalf("expected stateRepository type error, got %v", err)
	}
}

func TestResolvePackage_StateRepositoryURLMissing(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"stateRepository":{}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Fatalf("expected stateRepo.url error, got %v", err)
	}
}

func TestResolvePackage_StateRepositoryFull(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},
		"stateRepository":{"url":"git@x","ref":"main","cloneSecret":"sec","strategy":"kapp","path":"/state"}}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	sr := resolved.Spec.StateRepository
	if sr == nil || sr.URL != "git@x" || sr.CloneSecret != "sec" || sr.Strategy != "kapp" || sr.Path != "/state" {
		t.Fatalf("unexpected StateRepository: %+v", sr)
	}
	if sr.Ref != "main" {
		t.Fatalf("unexpected Ref: %q", sr.Ref)
	}
}

func TestResolvePackage_StateRepositoryRefWrongType(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"stateRepository":{"url":"git@x","ref":{"branch":"main"}}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "stateRepository: ref must be a string") {
		t.Fatalf("expected stateRepo.ref type error, got %v", err)
	}
}

func TestResolvePackage_MaintainersWrongType(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":"not-an-array"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "maintainers must be an array") {
		t.Fatalf("expected packageMaintainers type error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryNotObject(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":["not-an-object"]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "maintainers[0] must be an object") {
		t.Fatalf("expected maintainers[0] error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryMissingName(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":[{"email":"a@b.com"}]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "maintainers[0].name") {
		t.Fatalf("expected maintainers[0].name error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryMissingEmail(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":[{"name":"neo"}]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "maintainers[0].email") {
		t.Fatalf("expected maintainers[0].email error, got %v", err)
	}
}

func TestResolvePackage_MaintainersValid(t *testing.T) {
	p := pkgWithContract(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":[{"name":"neo","email":"neo@x.com"}]}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	if len(resolved.Spec.Maintainers) != 1 || resolved.Spec.Maintainers[0].Name != "neo" || resolved.Spec.Maintainers[0].Email != "neo@x.com" {
		t.Fatalf("unexpected Maintainers: %+v", resolved.Spec.Maintainers)
	}
}

func TestResolvePackage_ContractNilRawIsError(t *testing.T) {
	p := &environmentv1alpha1.Package{}
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "spec.contract is required") {
		t.Fatalf("expected spec.contract error, got %v", err)
	}
}

func TestResolvePackage_NameWrongType(t *testing.T) {
	p := pkgWithContract(`{"name":5,"version":"1.0.0","repository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

// TestResolvePackage_ContractKeysMatchTheProto resolves a contract written
// with the field names package.proto defines (the names the install samples
// and the docs use), and rejects the earlier package-prefixed spellings.
func TestResolvePackage_ContractKeysMatchTheProto(t *testing.T) {
	p := pkgWithContract(`{
		"enabled": true,
		"name": "for-kaniko-app",
		"version": "v1.2.3",
		"description": "manifests",
		"maintainers": [{"name": "Neo", "email": "neo@example.com"}],
		"repository": {"url": "git@github.com:example-org/packages.git", "credentialsSecret": "packages-creds"},
		"diffEnabled": true,
		"stateRepository": {
			"url": "git@github.com:example-org/state.git", "ref": "master",
			"cloneSecret": "state-creds", "strategy": "kustomization", "path": "./clusters/dev"
		}
	}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	spec := resolved.Spec
	if spec.Name != "for-kaniko-app" || spec.Version != "v1.2.3" || spec.Description != "manifests" || !spec.DiffEnabled {
		t.Errorf("unexpected scalars: %+v", spec)
	}
	if len(spec.Maintainers) != 1 || spec.Maintainers[0].Email != "neo@example.com" {
		t.Errorf("unexpected maintainers: %+v", spec.Maintainers)
	}
	if spec.PackageRepository.URL != "git@github.com:example-org/packages.git" || spec.PackageRepository.CredentialsSecret != "packages-creds" {
		t.Errorf("unexpected repository: %+v", spec.PackageRepository)
	}
	sr := spec.StateRepository
	if sr == nil || sr.URL != "git@github.com:example-org/state.git" || sr.Ref != "master" ||
		sr.CloneSecret != "state-creds" || sr.Strategy != "kustomization" || sr.Path != "./clusters/dev" {
		t.Errorf("unexpected stateRepository: %+v", sr)
	}

	legacy := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"}}`)
	if _, err := ResolvePackage(legacy); err == nil || !strings.Contains(err.Error(), `"name"`) {
		t.Errorf("legacy key names: error = %v, want the missing name field", err)
	}
}

func FuzzResolvePackage(f *testing.F) {
	f.Add(minimalValid)
	f.Add(`{not json`)
	f.Add(`{"name":5,"version":"1.0.0","repository":{"url":"x"}}`)
	f.Add(`{"name":"pkg1","version":"1.0.0","repository":"not-an-object"}`)
	f.Add(`{"name":"pkg1","version":"1.0.0","repository":{"url":"x"},"maintainers":[{"email":"a@b.com"}]}`)
	f.Add(`{"enabled":false,"name":"pkg1","version":"1.0.0","repository":{"url":"x"}}`)

	f.Fuzz(func(t *testing.T, raw string) {
		p := pkgWithContract(raw)
		_, _ = ResolvePackage(p)
	})
}
