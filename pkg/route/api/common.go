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
Package api — common.go

Shared utilities used across all Route Provider implementations.

TLS secret naming:

	certSecretName and sanitizeHost are the shared convention between
	the Route provider (DomainMapping.Spec.TLS.SecretName / Ingress.Spec.TLS)
	and the Domain provider (cert-manager Certificate secretName).
	Both sides must produce identical names — this file is the single source
	of truth. Never duplicate these functions in a provider file.
*/
package api

import "fmt"

// certSecretName returns the TLS Secret name for a given host.
// This convention is shared between:
//   - KnativeProvider  — DomainMapping.Spec.TLS.SecretName
//   - IngressProvider  — Ingress.Spec.TLS[].SecretName
//   - Domain provider  — cert-manager Certificate secretName
//
// All three must produce identical names for the TLS handshake to work.
func certSecretName(host string) string {
	return fmt.Sprintf("blanketops-tls-%s", sanitizeHost(host))
}

// sanitizeHost converts a hostname to a valid Kubernetes resource name segment
// by replacing dots with hyphens.
// e.g. api.dev.blanketops.online → api-dev-blanketops-online
func sanitizeHost(host string) string {
	result := make([]byte, len(host))
	for i := 0; i < len(host); i++ {
		if host[i] == '.' {
			result[i] = '-'
		} else {
			result[i] = host[i]
		}
	}
	return string(result)
}
