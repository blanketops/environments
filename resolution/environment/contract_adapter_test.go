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

package environment

import (
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
)

func TestToEnvironmentContract_NilSpec(t *testing.T) {
	var s *ResolvedEnvironmentSpec
	if got := s.ToEnvironmentContract(); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToEnvironmentContract_Minimal(t *testing.T) {
	s := &ResolvedEnvironmentSpec{ApplicationName: "app", EnvironmentType: "development"}
	out := s.ToEnvironmentContract()
	if out.ApplicationName != "app" {
		t.Fatalf("unexpected ApplicationName: %q", out.ApplicationName)
	}
	if out.Build != nil || out.GitRepository != nil || out.GitHubEvent != nil ||
		out.Deployment != nil || out.Route != nil || out.Package != nil || out.Contract != nil {
		t.Fatalf("expected all optional refs nil, got %+v", out)
	}
}

func TestToEnvironmentContract_AllRefsPopulated(t *testing.T) {
	s := &ResolvedEnvironmentSpec{
		ApplicationName: "app",
		Build:           "b1",
		GitRepository:   "gr1",
		GitHubEvent:     "ev1",
		ServiceUnits:    []string{"su1", "su2"},
		Deployment:      "d1",
		Route:           "r1",
		Package:         "p1",
		Contract:        &ResolvedEnvironmentContract{SecretStoreProvider: "vault"},
	}
	out := s.ToEnvironmentContract()
	if out.Build.Name != "b1" || out.GitRepository.Name != "gr1" || out.GitHubEvent.Name != "ev1" {
		t.Fatalf("unexpected refs: %+v", out)
	}
	if len(out.ServiceUnits) != 2 || out.ServiceUnits[0].Name != "su1" {
		t.Fatalf("unexpected ServiceUnits: %+v", out.ServiceUnits)
	}
	if out.Deployment.Name != "d1" || out.Route.Name != "r1" || out.Package.Name != "p1" {
		t.Fatalf("unexpected refs: %+v", out)
	}
	if out.Contract == nil || out.Contract.SecretStore.Provider.SecretStoreProvider != commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_VAULT {
		t.Fatalf("unexpected Contract: %+v", out.Contract)
	}
}

func TestToEnvironmentContract_ContractPresentButEmptyProvider(t *testing.T) {
	s := &ResolvedEnvironmentSpec{
		ApplicationName: "app",
		Contract:        &ResolvedEnvironmentContract{},
	}
	out := s.ToEnvironmentContract()
	if out.Contract != nil {
		t.Fatal("expected nil Contract when SecretStoreProvider is empty")
	}
}

func TestToEnvironmentTypeWrapper(t *testing.T) {
	cases := map[string]commoncontractv1.EnvironmentType_EnvironmentType{
		"development":              commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_DEVELOPMENT,
		"ENVIRONMENT_TYPE_STAGING": commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_STAGING,
		"production":               commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_PRODUCTION,
		"testing":                  commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_TESTING,
		"bogus":                    commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := toEnvironmentTypeWrapper(in); got.Type != want {
			t.Fatalf("toEnvironmentTypeWrapper(%q) = %v, want %v", in, got.Type, want)
		}
	}
}

func TestToSecretStoreProvider(t *testing.T) {
	cases := map[string]commoncontractv1.SecretStoreProvider_SecretStoreProvider{
		"aws":                         commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_AWS,
		"SECRET_STORE_PROVIDER_VAULT": commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_VAULT,
		"gcp":                         commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_GCP,
		"azure":                       commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_AZURE,
		"bogus":                       commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := toSecretStoreProvider(in); got.SecretStoreProvider != want {
			t.Fatalf("toSecretStoreProvider(%q) = %v, want %v", in, got.SecretStoreProvider, want)
		}
	}
}
