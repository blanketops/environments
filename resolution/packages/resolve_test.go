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

package packages

import (
	"context"
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

const minimalValid = `{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"oci://x"}}`

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
	p := pkgWithContract(`{"packageVersion":"1.0.0","packageRepository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageName") {
		t.Fatalf("expected packageName error, got %v", err)
	}
}

func TestResolvePackage_VersionMissing(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageRepository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageVersion") {
		t.Fatalf("expected packageVersion error, got %v", err)
	}
}

func TestResolvePackage_RepositoryMissing(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageRepository") {
		t.Fatalf("expected packageRepository error, got %v", err)
	}
}

func TestResolvePackage_RepositoryWrongType(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":"not-an-object"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), `field "packageRepository" must be an object`) {
		t.Fatalf("expected packageRepository type error, got %v", err)
	}
}

func TestResolvePackage_RepositoryURLMissing(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{}}`)
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
	if !resolved.Spec.Enabled {
		t.Fatal("expected Enabled to default to true")
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

func TestResolvePackage_EnabledExplicitFalse(t *testing.T) {
	p := pkgWithContract(`{"enabled":false,"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"}}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	if resolved.Spec.Enabled {
		t.Fatal("expected Enabled to be false")
	}
}

func TestResolvePackage_StateRepositoryWrongType(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"stateRepo":"not-an-object"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "stateRepo must be an object") {
		t.Fatalf("expected stateRepo type error, got %v", err)
	}
}

func TestResolvePackage_StateRepositoryURLMissing(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"stateRepo":{}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "url") {
		t.Fatalf("expected stateRepo.url error, got %v", err)
	}
}

func TestResolvePackage_StateRepositoryFull(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},
		"stateRepo":{"url":"git@x","ref":{"branch":"main","tag":"v1","commit":"abc"},"cloneSecret":"sec","strategy":"kapp","path":"/state"}}`)
	resolved, err := ResolvePackage(p)
	if err != nil {
		t.Fatalf("ResolvePackage: %v", err)
	}
	sr := resolved.Spec.StateRepository
	if sr == nil || sr.URL != "git@x" || sr.CloneSecret != "sec" || sr.Strategy != "kapp" || sr.Path != "/state" {
		t.Fatalf("unexpected StateRepository: %+v", sr)
	}
	if sr.Ref.Branch != "main" || sr.Ref.Tag != "v1" || sr.Ref.Commit != "abc" {
		t.Fatalf("unexpected Ref: %+v", sr.Ref)
	}
}

func TestResolvePackage_StateRepositoryRefWrongType(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"stateRepo":{"url":"git@x","ref":"not-an-object"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "stateRepo.ref must be an object") {
		t.Fatalf("expected stateRepo.ref type error, got %v", err)
	}
}

func TestResolvePackage_MaintainersWrongType(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"packageMaintainers":"not-an-array"}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageMaintainers must be an array") {
		t.Fatalf("expected packageMaintainers type error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryNotObject(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"packageMaintainers":["not-an-object"]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageMaintainers[0] must be an object") {
		t.Fatalf("expected packageMaintainers[0] error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryMissingName(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"packageMaintainers":[{"email":"a@b.com"}]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageMaintainers[0].name") {
		t.Fatalf("expected packageMaintainers[0].name error, got %v", err)
	}
}

func TestResolvePackage_MaintainersEntryMissingEmail(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"packageMaintainers":[{"name":"neo"}]}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "packageMaintainers[0].email") {
		t.Fatalf("expected packageMaintainers[0].email error, got %v", err)
	}
}

func TestResolvePackage_MaintainersValid(t *testing.T) {
	p := pkgWithContract(`{"packageName":"pkg1","packageVersion":"1.0.0","packageRepository":{"url":"x"},"packageMaintainers":[{"name":"neo","email":"neo@x.com"}]}`)
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
	p := pkgWithContract(`{"packageName":5,"packageVersion":"1.0.0","packageRepository":{"url":"x"}}`)
	_, err := ResolvePackage(p)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	p := pkgWithContract(minimalValid)
	resolved, err := a.Resolve(context.Background(), p)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Name != "pkg1" {
		t.Fatalf("unexpected Name: %q", resolved.Spec.Name)
	}
}
