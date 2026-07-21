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

// buildAzureProviderSpec builds the spec.provider.azurekv block of a
// ClusterSecretStore/SecretStore (ESO provider key "azurekv" — Azure Key
// Vault, not "azure").
//
// STUB: resolution/environment/resolve has no ResolvedAzureConfig yet —
// this needs the vaultUrl plus an authType (ManagedIdentity is the
// standard in-cluster pattern via workload identity, mirroring Vault's
// Kubernetes auth here; ServicePrincipal via a client ID/secret secretRef
// is the fallback). Wire the equivalent of resolve.go's
// contract.secretStore.vault parsing for contract.secretStore.azure before
// this can build a real spec.
func buildAzureProviderSpec(contract *environmentresolve.ResolvedEnvironmentContract) (map[string]any, error) {
	return nil, fmt.Errorf("secretStore provider \"azure\" is not yet implemented")
}
