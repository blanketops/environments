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
Package environment implements resolution for the Environment CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedEnvironmentSpec) to the protobuf contract type
(environmentscontractv1alpha1.EnvironmentSpec).

Controllers and domain logic MUST NOT consume the returned contract value.
The resolved runtime spec is the authoritative type for all reconciliation
decisions. The contract is a serialisation artifact, not a source of truth.

Key generated type facts:
  - SecretStoreProvider  → top-level enum (commoncontractv1.SecretStoreProvider).
    No wrapper message. SecretStore.Provider field takes the enum directly.
  - EnvironmentType      → wrapped enum (*commoncontractv1.EnvironmentType).
    Field carries EnvironmentType_EnvironmentType enum value.
  - All CR refs          → *environmentscontractv1alpha1.ObjectRef{Name: "..."}.
*/
package environment

import (
	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
	environmentscontractv1alpha1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

// ToEnvironmentContract projects the resolved runtime spec into a protobuf
// EnvironmentSpec for infrastructure consumers (hashing, comparison, audit).
//
// ⚠️ ONE-WAY adapter. MUST NOT be fed back into any controller or domain path.
func (s *ResolvedEnvironmentSpec) ToEnvironmentContract() *environmentscontractv1alpha1.EnvironmentSpec {
	if s == nil {
		return nil
	}

	spec := &environmentscontractv1alpha1.EnvironmentSpec{
		ApplicationName: s.ApplicationName,
		Branch:          s.Branch,
		GitOwner:        s.GitOwner,
		Version:         s.Version,
		Description:     s.Description,
		EnvironmentType: toEnvironmentTypeWrapper(s.EnvironmentType),
	}

	// ------------------------------------------------
	// Build ref (OPTIONAL).
	// ------------------------------------------------
	if s.Build != "" {
		spec.Build = &environmentscontractv1alpha1.ObjectRef{Name: s.Build}
	}

	// ------------------------------------------------
	// GitRepository ref (OPTIONAL).
	// The GitRepository CR owns branch, owner, and clone config.
	// ------------------------------------------------
	if s.GitRepository != "" {
		spec.GitRepository = &environmentscontractv1alpha1.ObjectRef{Name: s.GitRepository}
	}

	// ------------------------------------------------
	// GitHubEvent ref (OPTIONAL).
	// The GitHubEvent CR owns webhook event routing and Argo Events config.
	// ------------------------------------------------
	if s.GitHubEvent != "" {
		spec.GitHubEvent = &environmentscontractv1alpha1.ObjectRef{Name: s.GitHubEvent}
	}

	// ------------------------------------------------
	// ServiceUnit refs (OPTIONAL).
	// ------------------------------------------------
	for _, name := range s.ServiceUnits {
		spec.ServiceUnits = append(spec.ServiceUnits,
			&environmentscontractv1alpha1.ObjectRef{Name: name},
		)
	}

	// ------------------------------------------------
	// Deployment ref (OPTIONAL).
	// ------------------------------------------------
	if s.Deployment != "" {
		spec.Deployment = &environmentscontractv1alpha1.ObjectRef{Name: s.Deployment}
	}

	// ------------------------------------------------
	// Route ref (OPTIONAL).
	// ------------------------------------------------
	if s.Route != "" {
		spec.Route = &environmentscontractv1alpha1.ObjectRef{Name: s.Route}
	}

	// ------------------------------------------------
	// Package ref (OPTIONAL).
	// ------------------------------------------------
	if s.Package != "" {
		spec.Package = &environmentscontractv1alpha1.ObjectRef{Name: s.Package}
	}

	// ------------------------------------------------
	// Platform contract: secret store provider only.
	// SecretStoreProvider is a top-level enum — no wrapper message.
	// Each CR manages its own credential secrets independently.
	// Provider drives ESO ClusterSecretStore selection.
	// ------------------------------------------------
	if s.Contract != nil && s.Contract.SecretStoreProvider != "" {
		spec.Contract = &environmentscontractv1alpha1.EnvironmentContract{
			SecretStore: &commoncontractv1.SecretStore{
				Provider: toSecretStoreProvider(s.Contract.SecretStoreProvider),
			},
		}
	}

	return spec
}

// toEnvironmentTypeWrapper maps an environment type string to the
// corresponding *commoncontractv1.EnvironmentType wrapper message.
// EnvironmentType is a wrapped enum — field carries EnvironmentType_EnvironmentType.
// Unknown strings map to UNSPECIFIED — must not fail the projection.
func toEnvironmentTypeWrapper(t string) *commoncontractv1.EnvironmentType {
	switch t {
	case "ENVIRONMENT_TYPE_DEVELOPMENT", "development":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_DEVELOPMENT,
		}
	case "ENVIRONMENT_TYPE_STAGING", "staging":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_STAGING,
		}
	case "ENVIRONMENT_TYPE_PRODUCTION", "production":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_PRODUCTION,
		}
	case "ENVIRONMENT_TYPE_TESTING", "testing":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_TESTING,
		}
	default:
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_UNSPECIFIED,
		}
	}
}

// toSecretStoreProvider maps a provider string to the corresponding
// commoncontractv1.SecretStoreProvider enum value.
// SecretStoreProvider is a top-level enum — returned directly, no wrapper.
// Unknown strings map to UNSPECIFIED — must not fail the projection.
func toSecretStoreProvider(p string) *commoncontractv1.SecretStoreProvider {
	switch p {
	case "SECRET_STORE_PROVIDER_AWS", "aws":
		return &commoncontractv1.SecretStoreProvider{
			SecretStoreProvider: commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_AWS,
		}
	case "SECRET_STORE_PROVIDER_VAULT", "vault":
		return &commoncontractv1.SecretStoreProvider{
			SecretStoreProvider: commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_VAULT,
		}
	case "SECRET_STORE_PROVIDER_GCP", "gcp":
		return &commoncontractv1.SecretStoreProvider{
			SecretStoreProvider: commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_GCP,
		}
	case "SECRET_STORE_PROVIDER_AZURE", "azure":
		return &commoncontractv1.SecretStoreProvider{
			SecretStoreProvider: commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_AZURE,
		}
	default:
		return &commoncontractv1.SecretStoreProvider{
			SecretStoreProvider: commoncontractv1.SecretStoreProvider_SECRET_STORE_PROVIDER_UNSPECIFIED,
		}
	}
}
