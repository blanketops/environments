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
Package api builds the ESO SecretStore spec.provider.gcpsm block from ResolvedEnvironmentContract.

See also:
  - ./pkg/apis/environment/application/service.go — orchestrates reconciliation using this provider
*/
package api

import (
	"fmt"
	"strings"

	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

type gcpServiceAccountKeyRefConfig struct {
	SecretName      string
	SecretNamespace string
	SecretKey       string
}

type gcpStoreConfig struct {
	ProjectID            string
	ServiceAccountName   string
	ServiceAccountNS     string
	ServiceAccountKeyRef *gcpServiceAccountKeyRefConfig
}

// buildGCPProviderSpec builds the spec.provider.gcpsm block for an External
// Secrets Operator SecretStore or ClusterSecretStore.
//
// Supports Workload Identity via auth.workloadIdentity.serviceAccountRef as
// the primary in-cluster auth mode and auth.secretRef.secretRef (GCP service
// account JSON key) as the fallback, omitting auth when neither is set so ESO
// can fall back to ambient node/controller credentials.
func buildGCPProviderSpec(contract *environmentresolve.ResolvedEnvironmentContract) (map[string]any, error) {
	if contract == nil {
		return nil, fmt.Errorf("resolved environment contract is required")
	}

	cfg, err := extractGCPStoreConfig(contract)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, fmt.Errorf("gcp secretStore projectID is required")
	}

	gcpsmSpec := map[string]any{
		"projectID": cfg.ProjectID,
	}

	authSpec, err := buildGCPAuthSpec(cfg)
	if err != nil {
		return nil, err
	}
	if len(authSpec) > 0 {
		gcpsmSpec["auth"] = authSpec
	}

	return map[string]any{
		"gcpsm": gcpsmSpec,
	}, nil
}

func buildGCPAuthSpec(cfg gcpStoreConfig) (map[string]any, error) {
	hasWorkloadIdentity := strings.TrimSpace(cfg.ServiceAccountName) != ""
	hasSecretRef := cfg.ServiceAccountKeyRef != nil

	if hasWorkloadIdentity && hasSecretRef {
		return nil, fmt.Errorf("gcp secretStore cannot specify both workloadIdentity and secretRef")
	}

	if hasWorkloadIdentity {
		saRef := map[string]any{
			"name": cfg.ServiceAccountName,
		}
		if strings.TrimSpace(cfg.ServiceAccountNS) != "" {
			saRef["namespace"] = cfg.ServiceAccountNS
		}
		return map[string]any{
			"workloadIdentity": map[string]any{
				"serviceAccountRef": saRef,
			},
		}, nil
	}

	if hasSecretRef {
		refs := cfg.ServiceAccountKeyRef
		if strings.TrimSpace(refs.SecretName) == "" || strings.TrimSpace(refs.SecretKey) == "" {
			return nil, fmt.Errorf("gcp secretRef requires secret name and secret key")
		}

		secretRef := map[string]any{
			"name": refs.SecretName,
			"key":  refs.SecretKey,
		}
		if strings.TrimSpace(refs.SecretNamespace) != "" {
			secretRef["namespace"] = refs.SecretNamespace
		}

		return map[string]any{
			"secretRef": map[string]any{
				"secretRef": secretRef,
			},
		}, nil
	}

	// Omitting auth lets ESO fall back to ambient node/controller credentials.
	return nil, nil
}

// extractGCPStoreConfig bridges ResolvedEnvironmentContract to the GCP Secret
// Manager fields. Replace this body once contract.SecretStore.GCP is defined
// in resolution/environment/resolve.
func extractGCPStoreConfig(_ *environmentresolve.ResolvedEnvironmentContract) (gcpStoreConfig, error) {
	return gcpStoreConfig{}, fmt.Errorf("gcp secretStore resolution is not yet wired on ResolvedEnvironmentContract")
}

