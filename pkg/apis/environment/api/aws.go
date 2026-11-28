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
Package api builds the ESO SecretStore spec.provider.aws block from ResolvedEnvironmentContract.

See also:
  - ./pkg/apis/environment/application/service.go — orchestrates reconciliation using this provider
*/
package api

import (
	"fmt"

	environmentresolve "github.com/blanketops/environments/resolution/environment/resolve"
)

const defaultAWSService = "SecretsManager"

type awsSecretRefConfig struct {
	SecretName      string
	SecretNamespace string
	AccessKeyIDKey  string
	SecretAccessKey string
}

type awsStoreConfig struct {
	Region               string
	Service              string
	Role                 string
	ServiceAccountName   string
	ServiceAccountNS     string
	StaticCredentialRefs *awsSecretRefConfig
}

// buildAWSProviderSpec builds the spec.provider.aws block for an External
// Secrets Operator SecretStore or ClusterSecretStore.
//
// Supports IRSA via auth.jwt.serviceAccountRef as the primary in-cluster
// auth mode and auth.secretRef (accessKeyIDSecretRef /
// secretAccessKeySecretRef) as the fallback, omitting auth when neither is
// set so ESO can fall back to the controller's pod identity.
func buildAWSProviderSpec(contract *environmentresolve.ResolvedEnvironmentContract) (map[string]any, error) {
	if contract == nil {
		return nil, fmt.Errorf("resolved environment contract is required")
	}

	cfg, err := extractAWSStoreConfig(contract)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("aws secretStore region is required")
	}

	service := cfg.Service
	if service == "" {
		service = defaultAWSService
	}

	awsSpec := map[string]any{
		"service": service,
		"region":  cfg.Region,
	}

	if cfg.Role != "" {
		awsSpec["role"] = cfg.Role
	}

	authSpec, err := buildAWSAuthSpec(cfg)
	if err != nil {
		return nil, err
	}
	if len(authSpec) > 0 {
		awsSpec["auth"] = authSpec
	}

	return map[string]any{
		"aws": awsSpec,
	}, nil
}

func buildAWSAuthSpec(cfg awsStoreConfig) (map[string]any, error) {
	hasJWT := cfg.ServiceAccountName != ""
	hasStatic := cfg.StaticCredentialRefs != nil

	if hasJWT && hasStatic {
		return nil, fmt.Errorf("aws secretStore cannot specify both serviceAccountRef (jwt) and secretRef")
	}

	if hasJWT {
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

	if hasStatic {
		refs := cfg.StaticCredentialRefs
		if refs.SecretName == "" || refs.AccessKeyIDKey == "" || refs.SecretAccessKey == "" {
			return nil, fmt.Errorf("aws secretRef requires secret name, access key ID key, and secret access key")
		}

		accessKeyRef := map[string]any{
			"name": refs.SecretName,
			"key":  refs.AccessKeyIDKey,
		}
		secretKeyRef := map[string]any{
			"name": refs.SecretName,
			"key":  refs.SecretAccessKey,
		}
		if refs.SecretNamespace != "" {
			accessKeyRef["namespace"] = refs.SecretNamespace
			secretKeyRef["namespace"] = refs.SecretNamespace
		}

		return map[string]any{
			"secretRef": map[string]any{
				"accessKeyIDSecretRef":     accessKeyRef,
				"secretAccessKeySecretRef": secretKeyRef,
			},
		}, nil
	}

	// Omitting auth lets ESO use ambient pod/controller credentials.
	return nil, nil
}

// extractAWSStoreConfig bridges ResolvedEnvironmentContract to the AWS
// provider fields. Replace this body once contract.SecretStore.AWS is defined
// in resolution/environment/resolve.
func extractAWSStoreConfig(_ *environmentresolve.ResolvedEnvironmentContract) (awsStoreConfig, error) {
	return awsStoreConfig{}, fmt.Errorf("aws secretStore resolution is not yet wired on ResolvedEnvironmentContract")
}

