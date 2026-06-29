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
This file owns GitHubEventResult — the return type from the GitHubEvent
Provider and Observer. It captures the state of the webhook infrastructure
and the latest delivery record.

GitHubEventResult and GitHubEventStatus share the same fields — ToStatus()
converts between them for persistence. The distinction exists so the domain
logic operates on a result type and the Kubernetes layer operates on a status
type, keeping their concerns separate.
*/
package domain

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubEventResult is the unified return value for the GitHubEvent domain.
// It reflects both the infrastructure state (from the Provider) and the
// delivery state (from the Observer).
type GitHubEventResult struct {
	// Success is true only when the GitHubEvent completed successfully.
	// False when Triggered=true — the build has been dispatched, not confirmed.
	Success bool

	// Triggered is true when a BuildRun was created or already existed for
	// this execution hash.
	Triggered bool

	//Rejected bool

	PayloadRecieved bool

	// Phase represents the current lifecycle state of the GitHubEvent subscription.
	// e.g., "IngressEnsured", "PayloadReceived", "Failed".
	Phase string

	// Message is a human-readable explanation of the current state.
	Message string

	Logs []string
	// LastPayloadRef is the name of the most recent ephemeral GitHubPayload CR
	// minted by the Argo Sensor for this subscription.
	LastPayloadRef string

	// LastFailureAt is the timestamp of the most recent failure.
	// Populated by the buildrun observer for observability only.
	LastFailureAt *metav1.Time
}
