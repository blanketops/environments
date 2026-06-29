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
This file owns DomainResult — the outcome value returned by every Provider.Ensure
call. The application service (pkg/domain/application/service.go) receives this
and passes it to the status writer to update the Domain CR.

DomainResult is a plain value type — no pointers, no interfaces. Callers check
Provisioned() to determine whether the provider completed successfully before
acting on CertIssued or Phase.

See also:
  - pkg/domain/domain/state.go     — Phase constants carried in DomainResult
  - pkg/domain/api/provider.go     — Provider.Ensure returns DomainResult
  - pkg/domain/application/status.go — consumes DomainResult to write CR status
*/
package domain

// DomainResult is the outcome of a single Provider.Ensure call.
// Returned by every backend implementation.
type DomainResult struct {
	// Phase is the resulting lifecycle phase after the Ensure call.
	Phase Phase

	// CertIssued is true once a valid TLS certificate covers this host.
	// Always true for platform strategy once the DomainClaim is active.
	// True for custom strategy once the Certificate resource reaches Ready.
	CertIssued bool

	// Message carries human-readable detail on the outcome.
	// Always populated on failure; may be empty on success.
	Message string
}

// Provisioned returns true when the provider successfully completed the
// cert/mapping chain (Phase == PhaseReady). Callers use this as the success
// predicate before reading CertIssued.
func (r DomainResult) Provisioned() bool {
	return r.Phase == PhaseReady
}
