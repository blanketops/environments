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

package api

import (
	"fmt"

	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

// buildVaultProviderSpec builds the spec.provider.vault block of a
// ClusterSecretStore/SecretStore from the Environment-declared Vault
// connection. Vault authenticates via Kubernetes auth only, for now.
func buildVaultProviderSpec(vault *environmentresolve.ResolvedVaultConfig) (map[string]any, error) {
	if vault == nil {
		return nil, fmt.Errorf("contract declares provider \"vault\" but no vault config was resolved (resolver bug)")
	}

	auth := map[string]any{
		"mountPath": vault.MountPath,
		"role":      vault.Role,
	}
	if vault.ServiceAccountName != "" {
		auth["serviceAccountRef"] = map[string]any{
			"name": vault.ServiceAccountName,
		}
	}

	return map[string]any{
		"vault": map[string]any{
			"server":  vault.Address,
			"path":    vault.Path,
			"version": vault.Version,
			"auth": map[string]any{
				"kubernetes": auth,
			},
		},
	}, nil
}
