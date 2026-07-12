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
This file owns the Provider interface for the domain package — the contract
that all Domain backend implementations must satisfy.

A Provider is responsible for materializing the cert/mapping chain for a host.
What it emits depends on the TLSStrategy:

	platform: DomainClaim only — the platform DNS01 wildcard cert already covers
	          this host, so no Certificate work is needed.
	custom:   Issuer + DomainClaim + Certificate — HTTP01 ACME challenge via
	          the nginx solver in the tenant namespace.

Ensure is idempotent — calling it multiple times for the same Domain CR must
produce the same set of resources with no unintended side effects.

The concrete implementation lives in knative.go. Provider never writes to the
Domain CR status — that belongs to pkg/domain/application/status.go.

See also:
  - pkg/domain/api/knative.go          — Knative/cert-manager implementation
  - pkg/domain/application/backend_selector.go — selects Provider by TLSStrategy
  - pkg/domain/application/service.go  — calls Provider.Ensure and handles result
*/
package api

import (
	"context"

	"github.com/BlanketOps/environments/pkg/apis/domain/domain"
	domainResolution "github.com/BlanketOps/environments/resolution/domain"
)

// Provider materializes and maintains the cert/mapping chain for a Domain CR.
// Implementations must be idempotent and must not mutate the Domain CR directly.
type Provider interface {
	// Ensure provisions or reconciles the cert/mapping chain for the given
	// Domain. Returns PhaseReady when complete and PhaseFailed on terminal
	// failure. PhaseProvisioning is returned as a Go error (ErrCertProvisioning)
	// so the controller requeues while ACME challenge or cert issuance is
	// in progress.
	Ensure(ctx context.Context, resolved *domainResolution.ResolvedDomain, d domain.Domain) (domain.DomainResult, error)
}
