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

// GitHubEventResult is the unified return value for the GitHubEvent domain.
// It reflects both the infrastructure state (from the Provider) and the
// delivery state (from the Observer).
type GitHubEventResult struct {
	// Phase represents the current lifecycle state of the GitHubEvent subscription.
	// e.g., "IngressEnsured", "PayloadReceived", "Failed".
	Phase string
	// Message is a human-readable explanation of the current state.
	Message string
	// LastPayloadRef is the name of the most recent ephemeral GitHubPayload CR
	// minted by the Argo Sensor for this subscription.
	LastPayloadRef string
}

// Ensured returns a result indicating the Argo Events infrastructure
// (EventSource, Sensor, etc.) was successfully provisioned or verified.
// This is typically returned by the GitHubProvider.
func Ensured(msg string) GitHubEventResult {
	return GitHubEventResult{
		Phase:   "IngressEnsured",
		Message: msg,
	}
}

// PayloadReceived returns a result indicating a new GitHubPayload CR was
// observed by the controller, successfully closing the webhook delivery loop.
// This is typically returned by the GitHubEvent Observer.
func PayloadReceived(payloadRef string) GitHubEventResult {
	return GitHubEventResult{
		Phase:          "PayloadReceived",
		LastPayloadRef: payloadRef,
		Message:        "Latest payload delivery recorded",
	}
}

// ToStatus converts a GitHubEventResult to a GitHubEventStatus for persistence.
// The two types are structurally identical — the conversion is zero-cost.
func (r GitHubEventResult) ToStatus() GitHubEventResult {
	return GitHubEventResult{
		Phase:          r.Phase,
		Message:        r.Message,
		LastPayloadRef: r.LastPayloadRef,
	}
}

// Rejected returns a result indicating the infrastructure provisioning failed
// or the subscription is invalid.
func Rejected(msg string) GitHubEventResult {
	return GitHubEventResult{
		Phase:   "Failed",
		Message: msg,
	}
}

// Rejected returns a result indicating the infrastructure provisioning failed
// or the subscription is invalid.
func Accepted(msg string) GitHubEventResult {
	return GitHubEventResult{
		Phase:   "Passed",
		Message: msg,
	}
}
