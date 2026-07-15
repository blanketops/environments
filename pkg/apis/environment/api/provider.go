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

// Package api resolves the platform ClusterSecretStore name from the
// environment contract's secretStore provider declaration.
//
// Keys are platform constants — they never change regardless of provider.
// Only the ClusterSecretStore name changes. All composed CR secret
// reconcilers accept the store name as an injected parameter.
package api

const (
	// StoreAWS is the ClusterSecretStore name for AWS Secrets Manager via ESO.
	StoreAWS = "blanketops-environments-aws"
	// StoreVault is the ClusterSecretStore name for HashiCorp Vault via ESO.
	StoreVault = "blanketops-environments-vault"
	// StoreGCP is the ClusterSecretStore name for GCP Secret Manager via ESO.
	StoreGCP = "blanketops-environments-gcp"
	// StoreAzure is the ClusterSecretStore name for Azure Key Vault via ESO.
	StoreAzure = "blanketops-environments-azure"
	// StoreFake is the ClusterSecretStore name for the local kind fake store.
	StoreFake = "blanketops-environments-fake"
)

// ResolveStoreName maps the secretStore.provider string from the environment
// contract to the corresponding ClusterSecretStore name.
// Unknown or empty provider falls back to the fake store — safe for local
// kind cluster development. Must not be used in production without a valid
// provider declared on the Environment CR.
func ResolveStoreName(provider string) string {
	switch provider {
	case "aws", "SECRET_STORE_PROVIDER_AWS":
		return StoreAWS
	case "vault", "SECRET_STORE_PROVIDER_VAULT":
		return StoreVault
	case "gcp", "SECRET_STORE_PROVIDER_GCP":
		return StoreGCP
	case "azure", "SECRET_STORE_PROVIDER_AZURE":
		return StoreAzure
	default:
		return StoreFake
	}
}
