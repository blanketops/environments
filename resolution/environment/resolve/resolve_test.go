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

func envWithContract(raw string) *environmentv1alpha1.Environment {
	return &environmentv1alpha1.Environment{
		Spec: environmentv1alpha1.EnvironmentSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0"}`

func TestResolveEnvironment_NilEnvironment(t *testing.T) {
	resolved, err := ResolveEnvironment(nil)
	if err != nil {
		t.Fatalf("expected no error for nil environment, got %v", err)
	}
	if resolved == nil || resolved.Spec != nil {
		t.Fatalf("expected zero-value ResolvedEnvironment with nil Spec, got %+v", resolved)
	}
}

func TestResolveEnvironment_EmptyContract(t *testing.T) {
	e := &environmentv1alpha1.Environment{}
	if _, err := ResolveEnvironment(e); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveEnvironment_InvalidJSON(t *testing.T) {
	e := envWithContract(`{not json`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveEnvironment_RequiredFieldsMissing(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"applicationName", `{"branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0"}`},
		{"branch", `{"applicationName":"app","gitOwner":"me","environmentType":"development","version":"1.0.0"}`},
		{"gitOwner", `{"applicationName":"app","branch":"main","environmentType":"development","version":"1.0.0"}`},
		{"environmentType", `{"applicationName":"app","branch":"main","gitOwner":"me","version":"1.0.0"}`},
		{"version", `{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := envWithContract(tc.raw)
			if _, err := ResolveEnvironment(e); err == nil {
				t.Fatalf("expected error for missing %s", tc.name)
			}
		})
	}
}

func TestResolveEnvironment_MinimalValid(t *testing.T) {
	e := envWithContract(minimalValid)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.ApplicationName != "app" || resolved.Spec.Branch != "main" {
		t.Fatalf("unexpected spec: %+v", resolved.Spec)
	}
	if resolved.Spec.Build != "" || resolved.Spec.GitRepository != "" || resolved.Spec.Contract != nil {
		t.Fatalf("expected no optional refs resolved, got %+v", resolved.Spec)
	}
}

func TestResolveEnvironment_BuildRef(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","build":{"name":"my-build"}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Build != "my-build" {
		t.Fatalf("unexpected Build ref: %q", resolved.Spec.Build)
	}
}

func TestResolveEnvironment_BuildRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","build":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "build.name") {
		t.Fatalf("expected build.name error, got %v", err)
	}
}

func TestResolveEnvironment_GitRepositoryRef(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","gitRepository":{"name":"repo1"}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.GitRepository != "repo1" {
		t.Fatalf("unexpected GitRepository ref: %q", resolved.Spec.GitRepository)
	}
}

func TestResolveEnvironment_GitRepositoryRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","gitRepository":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "gitRepository.name") {
		t.Fatalf("expected gitRepository.name error, got %v", err)
	}
}

func TestResolveEnvironment_GitHubEventRef(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","gitHubEvent":{"name":"ev1"}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.GitHubEvent != "ev1" {
		t.Fatalf("unexpected GitHubEvent ref: %q", resolved.Spec.GitHubEvent)
	}
}

func TestResolveEnvironment_GitHubEventRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","gitHubEvent":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "gitHubEvent.name") {
		t.Fatalf("expected gitHubEvent.name error, got %v", err)
	}
}

func TestResolveEnvironment_ServiceUnits(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","serviceUnits":[{"name":"su1"},{"name":"su2"}]}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if len(resolved.Spec.ServiceUnits) != 2 || resolved.Spec.ServiceUnits[0] != "su1" || resolved.Spec.ServiceUnits[1] != "su2" {
		t.Fatalf("unexpected ServiceUnits: %+v", resolved.Spec.ServiceUnits)
	}
}

func TestResolveEnvironment_ServiceUnitsEntryNotObject(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","serviceUnits":["not-an-object"]}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "serviceUnits[0] must be an object") {
		t.Fatalf("expected serviceUnits[0] error, got %v", err)
	}
}

func TestResolveEnvironment_ServiceUnitsEntryMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","serviceUnits":[{}]}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "serviceUnits[0].name") {
		t.Fatalf("expected serviceUnits[0].name error, got %v", err)
	}
}

func TestResolveEnvironment_DeploymentRouteDomainPackageRefs(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"deployment":{"name":"depl1"},"route":{"name":"route1"},"package":{"name":"pkg1"}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Deployment != "depl1" || resolved.Spec.Route != "route1" || resolved.Spec.Package != "pkg1" {
		t.Fatalf("unexpected refs: %+v", resolved.Spec)
	}
}

// This test documents an existing bug rather than desired behavior: the
// "domain" branch in resolve.go assigns to spec.Route instead of a
// spec.Domain field (which doesn't exist on ResolvedEnvironmentSpec at all).
// Declaring "domain" silently overwrites whatever "route" resolved to.
func TestResolveEnvironment_DomainRefOverwritesRoute_KnownBug(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"route":{"name":"route1"},"domain":{"name":"domain1"}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Route != "domain1" {
		t.Fatalf("expected the known domain->Route overwrite bug (got %q) — if this now reads %q, the bug was fixed and this test should be updated",
			resolved.Spec.Route, "route1")
	}
}

func TestResolveEnvironment_DeploymentRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","deployment":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "deployment.name") {
		t.Fatalf("expected deployment.name error, got %v", err)
	}
}

func TestResolveEnvironment_RouteRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","route":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "route.name") {
		t.Fatalf("expected route.name error, got %v", err)
	}
}

func TestResolveEnvironment_DomainRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","domain":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "domain.name") {
		t.Fatalf("expected domain.name error, got %v", err)
	}
}

func TestResolveEnvironment_PackageRefMissingName(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","package":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "package.name") {
		t.Fatalf("expected package.name error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStore(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","path":"secret","role":"environments"}}}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Contract == nil || resolved.Spec.Contract.SecretStoreProvider != "vault" {
		t.Fatalf("unexpected Contract: %+v", resolved.Spec.Contract)
	}
	vault := resolved.Spec.Contract.Vault
	if vault == nil {
		t.Fatal("expected non-nil Vault config")
	}
	if vault.Address != "https://vault:8200" || vault.Path != "secret" || vault.Role != "environments" {
		t.Fatalf("unexpected Vault config: %+v", vault)
	}
	if vault.MountPath != "kubernetes" {
		t.Fatalf("expected default mountPath %q, got %q", "kubernetes", vault.MountPath)
	}
	if vault.Version != "v2" {
		t.Fatalf("expected default version %q, got %q", "v2", vault.Version)
	}
	if vault.ServiceAccountName != "" {
		t.Fatalf("expected empty ServiceAccountName, got %q", vault.ServiceAccountName)
	}
}

func TestResolveEnvironment_ContractSecretStoreVaultOverrides(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","path":"secret","role":"environments","mountPath":"eso-kubernetes","version":"v1","serviceAccountName":"external-secrets"}}}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	vault := resolved.Spec.Contract.Vault
	if vault.MountPath != "eso-kubernetes" || vault.Version != "v1" || vault.ServiceAccountName != "external-secrets" {
		t.Fatalf("unexpected Vault config overrides: %+v", vault)
	}
}

func TestResolveEnvironment_ContractSecretStoreVaultMissing(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault"}}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore.vault is required") {
		t.Fatalf("expected contract.secretStore.vault error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStoreVaultMissingAddress(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault","vault":{"path":"secret","role":"environments"}}}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore.vault.address") {
		t.Fatalf("expected vault.address error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStoreVaultMissingPath(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","role":"environments"}}}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore.vault.path") {
		t.Fatalf("expected vault.path error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStoreVaultMissingRole(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","path":"secret"}}}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore.vault.role") {
		t.Fatalf("expected vault.role error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStoreNonVaultProviderNoVaultRequired(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0",
		"contract":{"secretStore":{"provider":"aws"}}}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Contract.Vault != nil {
		t.Fatalf("expected nil Vault config for aws provider, got %+v", resolved.Spec.Contract.Vault)
	}
}

func TestResolveEnvironment_ContractMissingSecretStore(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore is required") {
		t.Fatalf("expected contract.secretStore error, got %v", err)
	}
}

func TestResolveEnvironment_ContractSecretStoreMissingProvider(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{"secretStore":{}}}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "contract.secretStore.provider") {
		t.Fatalf("expected contract.secretStore.provider error, got %v", err)
	}
}

func TestResolveEnvironment_ApplicationNameWrongType(t *testing.T) {
	e := envWithContract(`{"applicationName":5,"branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0"}`)
	_, err := ResolveEnvironment(e)
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

func TestResolveEnvironment_DescriptionValid(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","description":"a test env"}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Description != "a test env" {
		t.Fatalf("unexpected Description: %q", resolved.Spec.Description)
	}
}

func TestResolveEnvironment_DescriptionWrongType(t *testing.T) {
	e := envWithContract(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","description":5}`)
	resolved, err := ResolveEnvironment(e)
	if err != nil {
		t.Fatalf("ResolveEnvironment: %v", err)
	}
	if resolved.Spec.Description != "" {
		t.Fatalf("expected empty Description for wrong type, got %q", resolved.Spec.Description)
	}
}

func FuzzResolveEnvironment(f *testing.F) {
	f.Add(minimalValid)
	f.Add(`{not json`)
	f.Add(`{"applicationName":5,"branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0"}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","build":{}}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{}}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{"secretStore":{}}}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{"secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","path":"secret","role":"environments"}}}}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","contract":{"secretStore":{"provider":"vault","vault":"not-an-object"}}}`)
	f.Add(`{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","serviceUnits":[{}]}`)

	f.Fuzz(func(t *testing.T, raw string) {
		e := envWithContract(raw)
		_, _ = ResolveEnvironment(e)
	})
}
