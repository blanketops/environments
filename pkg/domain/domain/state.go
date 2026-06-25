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
This file owns the Phase type and its constants for the Domain domain layer.

Phase mirrors blanketops.common.v1.DomainPhase in the proto contract. The
status writer (pkg/domain/application/status.go) maps Phase values to the
conditions on the Domain CR status subresource.

See also:
  - pkg/domain/domain/result.go    — DomainResult carries Phase as its outcome
  - pkg/domain/application/status.go — writes Phase to the CR status
*/
package domain

// Phase represents the current lifecycle phase of a Domain CR.
// Set by the status writer after each provider Ensure call.
type Phase string

const (
	// PhasePending indicates the controller has accepted the Domain CR but
	// cert/mapping work has not yet started.
	PhasePending Phase = "Pending"

	// PhaseProvisioning indicates cert issuance or DomainClaim creation is
	// in progress. The controller will requeue until this resolves.
	PhaseProvisioning Phase = "Provisioning"

	// PhaseReady indicates the cert is issued (or covered by the platform
	// wildcard), the DomainClaim is active, and the host is reachable.
	PhaseReady Phase = "Ready"

	// PhaseFailed indicates a terminal failure — cert issuance failed,
	// DomainClaim was rejected, or an ACME HTTP01 challenge could not be
	// satisfied. Check DomainResult.Message and the CR conditions for detail.
	PhaseFailed Phase = "Failed"
)
