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
Package api builds the ESO SecretStore spec.provider.azurekv block from ResolvedEnvironmentContract.

See also:
  - ./pkg/apis/environment/application/service.go — orchestrates reconciliation using this provider
*/
package api

import (
	"fmt"
	"strings"

	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

type azureServicePrincipalConfig struct {
	SecretName      string
	SecretNamespace string
	ClientIDKey     string
	ClientSecretKey string
}

type azureStoreConfig struct {
	VaultURL             string
	TenantID             string
	EnvironmentType      string
	ServiceAccountName   string
	ServiceAccountNS     string
	ServicePrincipalRefs *azureServicePrincipalConfig
}

// buildAzureProviderSpec builds the spec.provider.azurekv block for an
// External Secrets Operator SecretStore or ClusterSecretStore.
//
// Supports Workload Identity via auth.jwt.serviceAccountRef as the primary
// in-cluster auth mode and auth.secretRef (clientId / clientSecret) for
// service principal credentials, omitting auth when neither is set so ESO
// can fall back to ambient pod/controller managed identity.
func buildAzureProviderSpec(contract *environmentresolve.ResolvedEnvironmentContract) (map[string]any, error) {
	if contract == nil {
		return nil, fmt.Errorf("resolved environment contract is required")
	}

	cfg, err := extractAzureStoreConfig(contract)
	if err != nil {
		return nil, err
	}
	if cfg.VaultURL == "" {
		return nil, fmt.Errorf("azure secretStore vaultURL is required")
	}

	azureKVSpec := map[string]any{
		"vaultUrl": cfg.VaultURL,
	}

	if cfg.TenantID != "" {
		azureKVSpec["tenantId"] = cfg.TenantID
	}
	if cfg.EnvironmentType != "" {
		azureKVSpec["environmentType"] = cfg.EnvironmentType
	}

	authSpec, err := buildAzureAuthSpec(cfg)
	if err != nil {
		return nil, err
	}
	if len(authSpec) > 0 {
		azureKVSpec["auth"] = authSpec
	}

	return map[string]any{
		"azurekv": azureKVSpec,
	}, nil
}

func buildAzureAuthSpec(cfg azureStoreConfig) (map[string]any, error) {
	hasWorkloadIdentity := cfg.ServiceAccountName != ""
	hasServicePrincipal := cfg.ServicePrincipalRefs != nil

	if hasWorkloadIdentity && hasServicePrincipal {
		return nil, fmt.Errorf("azure secretStore cannot specify both serviceAccountRef (workload identity) and secretRef")
	}

	if hasWorkloadIdentity {
		saRef := map[string]any{
			"name": cfg.ServiceAccountName,
		}
		if cfg.ServiceAccountNS != "" {
			saRef["namespace"] = cfg.ServiceAccountNS
		}
		return map[string]any{
			"jwt": map[string]any{
				"serviceAccountRef": saRef,
			},
		}, nil
	}

	if hasServicePrincipal {
		refs := cfg.ServicePrincipalRefs
		if strings.TrimSpace(refs.SecretName) == "" ||
			strings.TrimSpace(refs.ClientIDKey) == "" ||
			strings.TrimSpace(refs.ClientSecretKey) == "" {
			return nil, fmt.Errorf("azure secretRef requires secret name, client ID key, and client secret key")
		}

		clientIDRef := map[string]any{
			"name": refs.SecretName,
			"key":  refs.ClientIDKey,
		}
		clientSecretRef := map[string]any{
			"name": refs.SecretName,
			"key":  refs.ClientSecretKey,
		}
		if refs.SecretNamespace != "" {
			clientIDRef["namespace"] = refs.SecretNamespace
			clientSecretRef["namespace"] = refs.SecretNamespace
		}

		return map[string]any{
			"secretRef": map[string]any{
				"clientId":     clientIDRef,
				"clientSecret": clientSecretRef,
			},
		}, nil
	}

	// Omitting auth lets ESO fall back to ambient pod/controller managed identity.
	return nil, nil
}

// extractAzureStoreConfig bridges ResolvedEnvironmentContract to the Azure Key
// Vault provider fields. Replace this body once contract.SecretStore.Azure is
// defined in resolution/environment/resolve.
func extractAzureStoreConfig(_ *environmentresolve.ResolvedEnvironmentContract) (azureStoreConfig, error) {
	return azureStoreConfig{}, fmt.Errorf("azure secretStore resolution is not yet wired on ResolvedEnvironmentContract")
}

