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
This file owns GitHubEventResult — the return type from every GitHubEvent
Provider, and the three constructor functions (Accepted, Triggered, Rejected)
that cover the common outcomes.

GitHubEventResult and GitHubEventStatus share the same fields — ToStatus()
converts between them for persistence. The distinction exists so the provider
layer operates on a result type and the status layer operates on a status type,
keeping their concerns separate.
*/
package domain

// GitHubEventResult is the unified return value from a GitHubEvent Provider.
type GitHubEventResult struct {
	// Accepted indicates the event passed validation and was admitted.
	Accepted bool
	// Triggered indicates the event caused downstream work to be dispatched.
	Triggered bool
	// Message is a human-readable explanation of the outcome.
	Message string
	// TriggeredRef is the name of the downstream resource created, if any.
	TriggeredRef string
}

// ToStatus converts a GitHubEventResult to a GitHubEventStatus for persistence.
// The two types are structurally identical — the conversion is zero-cost.
func (r GitHubEventResult) ToStatus() GitHubEventStatus {
	return GitHubEventStatus(r)
}

// Accepted returns a result indicating the event was admitted but did not
// trigger downstream work.
func Accepted(msg string) GitHubEventResult {
	return GitHubEventResult{Accepted: true, Message: msg}
}

// Triggered returns a result indicating the event was admitted and caused
// downstream work to be dispatched to ref.
func Triggered(ref, msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted:     true,
		Triggered:    true,
		TriggeredRef: ref,
		Message:      msg,
	}
}

// Rejected returns a result indicating the event was not admitted.
func Rejected(msg string) GitHubEventResult {
	return GitHubEventResult{Accepted: false, Message: msg}
}
