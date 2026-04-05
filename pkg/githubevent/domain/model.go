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

package domain

// EventType is the semantic type of a GitHub event.
type EventType string
type EventID string

const (
	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
)

// GitRef represents a Git reference.
type GitRef struct {
	Name string // e.g. refs/heads/main
}

// Commit represents a commit involved in the event.
type Commit struct {
	SHA string
}

// Actor represents the identity that triggered the event.
type Actor struct {
	Login string
}

// Repository identifies the source repository.
type Repository struct {
	FullName string // owner/name
	Owner    string
	Name     string
}

// GitHubEvent is a pure domain fact:
// “Something happened in a repository”.
type GitHubEvent struct {
	EventID    string
	Type       EventType
	Repository Repository
	Ref        GitRef
	Commit     Commit
	Actor      Actor
}

// -----------------------------------------------------------------------------
// GitHubEventStatus
// -----------------------------------------------------------------------------
//
// Internal domain status for a GitHubEvent decision.
// This is serialized into CRD status.contract.
type GitHubEventStatus struct {
	// Was the event accepted as valid?
	Accepted bool

	// Did this event trigger downstream work?
	Triggered bool

	// Human-readable explanation
	Message string

	// Reference to triggered resource (optional)
	TriggeredRef string
}
