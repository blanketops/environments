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
Package resolve implements resolution for the Environment CR.

The Environment CR stores its spec as a raw JSON contract (spec.contract).
ResolveEnvironment decodes this into a fully typed ResolvedEnvironment —
the authoritative runtime representation consumed by all downstream domain,
contract projection, and application logic.

Resolution is nil-tolerant for the CR itself but strict on the contract
payload — a missing or malformed contract, or an unsupported/malformed
secretStore provider configuration, is always an error. All failures
surface as errors. Resolution never panics.
*/
package resolve

import (
	"encoding/json"
	"fmt"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
)

// -----------------------------------------------------------------------------
// Runtime types (AUTHORITATIVE)
// -----------------------------------------------------------------------------

// ResolvedEnvironment pairs the original Kubernetes Environment object with
// its fully decoded and validated spec.
type ResolvedEnvironment struct {
	Environment *environmentv1alpha1.Environment
	Spec        *ResolvedEnvironmentSpec
}

// ResolvedEnvironmentSpec is the decoded Environment spec.
// All CR references are stored as name strings — cross-CR lookups are the
// responsibility of the domain layer.
type ResolvedEnvironmentSpec struct {
	ApplicationName string
	Branch          string
	GitOwner        string
	EnvironmentType string
	Version         string
	Description     string

	// Optional CR references — empty string means not declared.
	Build         string
	GitRepository string
	GitHubEvent   string
	Deployment    string
	Route         string
	Domain        string
	Package       string

	// Slices are nil when not declared; empty slice is never returned.
	ServiceUnits []string

	// Platform-level bindings.
	Contract *ResolvedEnvironmentContract
}

// ResolvedEnvironmentContract holds platform-level bindings declared on the
// Environment — the secret store provider selection and provider-specific
// configuration for the selected backend.
type ResolvedEnvironmentContract struct {
	SecretStore ResolvedSecretStore
}

// ResolvedSecretStore holds the selected secret store provider, optional
// store override name, and the resolved configuration for that provider.
// At most one provider config pointer is non-nil for a valid resolved store.
type ResolvedSecretStore struct {
	Provider  string
	StoreName string
	Vault     *ResolvedVaultConfig
	AWS       *ResolvedAWSConfig
	Azure     *ResolvedAzureConfig
	GCP       *ResolvedGCPConfig
}

// ResolvedVaultConfig is the Environment-declared HashiCorp Vault connection
// used to provision the ClusterSecretStore/SecretStore ESO reads from.
// This is connection config only — secret values are populated into Vault
// by a separate out-of-band process, never by this repo.
type ResolvedVaultConfig struct {
	// Address is the Vault server URL. Required.
	Address string
	// Path is the KV secrets engine mount path (e.g. "secret"). Required.
	Path string
	// Role is the Vault Kubernetes-auth role to assume. Required.
	Role string
	// MountPath is the Vault Kubernetes-auth mount path. Defaults to
	// "kubernetes" when not declared.
	MountPath string
	// Version is the KV engine version ("v1" or "v2"). Defaults to "v2"
	// when not declared.
	Version string
	// ServiceAccountName optionally names the ServiceAccount ESO uses to
	// authenticate the Kubernetes-auth login. Empty means ESO falls back
	// to its own pod identity.
	ServiceAccountName string
}

// ResolvedAWSConfig is the Environment-declared AWS Secrets Manager /
// Parameter Store configuration for the backing secret store.
type ResolvedAWSConfig struct {
	// Region is the AWS region (e.g. "af-south-1", "us-east-1"). Required.
	Region string
	// Service is the AWS service name. Defaults to "SecretsManager" when
	// not declared.
	Service string
	// Role optionally specifies an IAM role ARN to assume.
	Role string
	// ServiceAccountName optionally names the ServiceAccount used for
	// workload/JWT authentication.
	ServiceAccountName string
	// ServiceAccountNamespace optionally specifies the namespace of the
	// ServiceAccount referenced above.
	ServiceAccountNamespace string
}

// ResolvedAzureConfig is the Environment-declared Azure Key Vault
// configuration for the backing secret store.
type ResolvedAzureConfig struct {
	// VaultURL is the Azure Key Vault URI (e.g. "https://my-vault.vault.azure.net"). Required.
	VaultURL string
	// TenantID optionally specifies the Azure AD / Entra ID tenant ID.
	TenantID string
	// EnvironmentType optionally specifies the Azure cloud environment
	// (e.g. "PublicCloud").
	EnvironmentType string
	// ServiceAccountName optionally names the ServiceAccount used for
	// workload identity / JWT authentication.
	ServiceAccountName string
	// ServiceAccountNamespace optionally specifies the namespace of the
	// ServiceAccount referenced above.
	ServiceAccountNamespace string
}

// ResolvedGCPConfig is the Environment-declared GCP Secret Manager
// configuration for the backing secret store.
type ResolvedGCPConfig struct {
	// ProjectID is the GCP project ID owning the secrets. Required.
	ProjectID string
	// ServiceAccountName optionally names the ServiceAccount used for
	// workload identity authentication.
	ServiceAccountName string
	// ServiceAccountNamespace optionally specifies the namespace of the
	// ServiceAccount referenced above.
	ServiceAccountNamespace string
}

// -----------------------------------------------------------------------------
// Resolution entry point
// -----------------------------------------------------------------------------

// ResolveEnvironment decodes and validates the raw JSON contract from the
// Environment CR spec into a ResolvedEnvironment.
func ResolveEnvironment(environment *environmentv1alpha1.Environment) (*ResolvedEnvironment, error) {
	if environment == nil {
		return &ResolvedEnvironment{}, nil
	}

	if len(environment.Spec.Contract.Raw) == 0 {
		return nil, fmt.Errorf("spec.contract is required")
	}

	var raw map[string]any
	if err := json.Unmarshal(environment.Spec.Contract.Raw, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode raw contract: %w", err)
	}

	// ------------------------------------------------
	// Required scalar fields.
	// ------------------------------------------------
	appName, err := mustString(raw, "applicationName")
	if err != nil {
		return nil, err
	}

	branch, err := mustString(raw, "branch")
	if err != nil {
		return nil, err
	}

	gitOwner, err := mustString(raw, "gitOwner")
	if err != nil {
		return nil, err
	}

	envType, err := mustString(raw, "environmentType")
	if err != nil {
		return nil, err
	}

	version, err := mustString(raw, "version")
	if err != nil {
		return nil, err
	}

	spec := &ResolvedEnvironmentSpec{
		ApplicationName: appName,
		Branch:          branch,
		GitOwner:        gitOwner,
		EnvironmentType: envType,
		Version:         version,
		Description:     optionalString(raw, "description"),
	}

	// ------------------------------------------------
	// Optional CR reference: build.
	// ------------------------------------------------
	if v, ok := raw["build"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("build.name: %w", err)
		}
		spec.Build = n
	}

	// ------------------------------------------------
	// Optional CR reference: gitRepository.
	// The GitRepository CR owns branch, owner, and clone config.
	// ------------------------------------------------
	if v, ok := raw["gitRepository"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("gitRepository.name: %w", err)
		}
		spec.GitRepository = n
	}

	// ------------------------------------------------
	// Optional CR reference: gitHubEvent.
	// The GitHubEvent CR owns webhook event routing and Argo Events config.
	// ------------------------------------------------
	if v, ok := raw["gitHubEvent"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("gitHubEvent.name: %w", err)
		}
		spec.GitHubEvent = n
	}

	// ------------------------------------------------
	// Optional CR reference list: serviceUnits.
	// ------------------------------------------------
	if list, ok := raw["serviceUnits"].([]any); ok {
		for i, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("serviceUnits[%d] must be an object", i)
			}
			n, err := mustString(m, "name")
			if err != nil {
				return nil, fmt.Errorf("serviceUnits[%d].name: %w", i, err)
			}
			spec.ServiceUnits = append(spec.ServiceUnits, n)
		}
	}

	// ------------------------------------------------
	// Optional CR references: deployment, route, domain, package.
	// ------------------------------------------------
	if v, ok := raw["deployment"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("deployment.name: %w", err)
		}
		spec.Deployment = n
	}

	if v, ok := raw["route"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("route.name: %w", err)
		}
		spec.Route = n
	}

	if v, ok := raw["domain"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("domain.name: %w", err)
		}
		spec.Domain = n
	}

	if v, ok := raw["package"].(map[string]any); ok {
		n, err := mustString(v, "name")
		if err != nil {
			return nil, fmt.Errorf("package.name: %w", err)
		}
		spec.Package = n
	}

	// ------------------------------------------------
	// Platform contract: secretStore is required when contract is declared.
	// ------------------------------------------------
	ssRaw, ok := raw["secretStore"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("contract.secretStore is required")
	}
	provider, err := mustString(ssRaw, "provider")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.provider: %w", err)
	}
	storeName := optionalString(ssRaw, "storeName")

	secretStore := ResolvedSecretStore{
		Provider:  provider,
		StoreName: storeName,
	}

	// --------------------------------------------
	// Provider-specific connection config — resolved
	// strictly per provider so every valid contract
	// carries typed config for contract projection and
	// api-level store reconciliation.
	// --------------------------------------------
	switch provider {
	case "vault", "SECRET_STORE_PROVIDER_VAULT":
		vaultCfg, err := resolveVaultConfig(ssRaw, provider)
		if err != nil {
			return nil, err
		}
		secretStore.Vault = vaultCfg

	case "aws", "SECRET_STORE_PROVIDER_AWS":
		awsCfg, err := resolveAWSConfig(ssRaw, provider)
		if err != nil {
			return nil, err
		}
		secretStore.AWS = awsCfg

	case "azure", "SECRET_STORE_PROVIDER_AZURE":
		azureCfg, err := resolveAzureConfig(ssRaw, provider)
		if err != nil {
			return nil, err
		}
		secretStore.Azure = azureCfg

	case "gcp", "SECRET_STORE_PROVIDER_GCP":
		gcpCfg, err := resolveGCPConfig(ssRaw, provider)
		if err != nil {
			return nil, err
		}
		secretStore.GCP = gcpCfg

	default:
		return nil, fmt.Errorf("unsupported contract.secretStore.provider %q", provider)
	}

	spec.Contract = &ResolvedEnvironmentContract{
		SecretStore: secretStore,
	}

	return &ResolvedEnvironment{
		Environment: environment,
		Spec:        spec,
	}, nil
}

// -----------------------------------------------------------------------------
// Per-provider secretStore resolvers
// -----------------------------------------------------------------------------

func resolveVaultConfig(ssRaw map[string]any, provider string) (*ResolvedVaultConfig, error) {
	vaultRaw, ok := ssRaw["vault"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("contract.secretStore.vault is required for provider %q", provider)
	}
	address, err := mustString(vaultRaw, "address")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.vault.address: %w", err)
	}
	path, err := mustString(vaultRaw, "path")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.vault.path: %w", err)
	}
	role, err := mustString(vaultRaw, "role")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.vault.role: %w", err)
	}

	mountPath := optionalString(vaultRaw, "mountPath")
	if mountPath == "" {
		mountPath = "kubernetes"
	}
	version := optionalString(vaultRaw, "version")
	if version == "" {
		version = "v2"
	}

	return &ResolvedVaultConfig{
		Address:            address,
		Path:               path,
		Role:               role,
		MountPath:          mountPath,
		Version:            version,
		ServiceAccountName: optionalString(vaultRaw, "serviceAccountName"),
	}, nil
}

func resolveAWSConfig(ssRaw map[string]any, provider string) (*ResolvedAWSConfig, error) {
	awsRaw, ok := ssRaw["aws"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("contract.secretStore.aws is required for provider %q", provider)
	}
	region, err := mustString(awsRaw, "region")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.aws.region: %w", err)
	}

	service := optionalString(awsRaw, "service")
	if service == "" {
		service = "SecretsManager"
	}

	return &ResolvedAWSConfig{
		Region:                  region,
		Service:                 service,
		Role:                    optionalString(awsRaw, "role"),
		ServiceAccountName:      optionalString(awsRaw, "serviceAccountName"),
		ServiceAccountNamespace: optionalString(awsRaw, "serviceAccountNamespace"),
	}, nil
}

func resolveAzureConfig(ssRaw map[string]any, provider string) (*ResolvedAzureConfig, error) {
	azureRaw, ok := ssRaw["azure"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("contract.secretStore.azure is required for provider %q", provider)
	}
	vaultURL, err := mustString(azureRaw, "vaultUrl")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.azure.vaultUrl: %w", err)
	}

	return &ResolvedAzureConfig{
		VaultURL:                vaultURL,
		TenantID:                optionalString(azureRaw, "tenantId"),
		EnvironmentType:         optionalString(azureRaw, "environmentType"),
		ServiceAccountName:      optionalString(azureRaw, "serviceAccountName"),
		ServiceAccountNamespace: optionalString(azureRaw, "serviceAccountNamespace"),
	}, nil
}

func resolveGCPConfig(ssRaw map[string]any, provider string) (*ResolvedGCPConfig, error) {
	gcpRaw, ok := ssRaw["gcp"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("contract.secretStore.gcp is required for provider %q", provider)
	}
	projectID, err := mustString(gcpRaw, "projectId")
	if err != nil {
		return nil, fmt.Errorf("contract.secretStore.gcp.projectId: %w", err)
	}

	return &ResolvedGCPConfig{
		ProjectID:               projectID,
		ServiceAccountName:      optionalString(gcpRaw, "serviceAccountName"),
		ServiceAccountNamespace: optionalString(gcpRaw, "serviceAccountNamespace"),
	}, nil
}

// -----------------------------------------------------------------------------
// Extraction helpers
// -----------------------------------------------------------------------------

func mustString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("field %q must be a non-empty string", key)
	}
	return s, nil
}

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
