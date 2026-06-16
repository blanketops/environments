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
This file owns the core domain types for the GitHubEvent domain — GitHubEvent,
Repository, GitRef, Commit, Actor, and GitHubEventStatus.

GitHubEvent is a pure domain fact: "something happened in a repository." It
carries no Kubernetes object references and no policy — those belong to the
provider and application layers respectively.

GitHubEventStatus is serialised into the CR's status.contract field by the
StatusWriter. It is the authoritative machine-readable outcome consumed by
downstream automation.
*/
package domain

// EventType is the semantic type of a GitHub webhook event.
type EventType string

// EventID is the provider-assigned delivery GUID.
type EventID string

const (
	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
)

// GitRef represents a Git reference carried by the event.
type GitRef struct {
	Name string // e.g. refs/heads/main, refs/tags/v1.0.0
}

// Commit represents the commit at the head of the event.
type Commit struct {
	SHA string
}

// Actor is the identity that triggered the event on the provider.
type Actor struct {
	Login string
}

// Repository identifies the source repository on the provider.
type Repository struct {
	FullName string // "owner/name" — the canonical identifier
	Owner    string
	Name     string
}

// GitHubEvent is the pure domain representation of an observed GitHub webhook
// event. It is produced by the Mapper and consumed by the Provider.
type GitHubEvent struct {
	EventID    string
	Type       EventType
	Repository Repository
	Ref        GitRef
	Commit     Commit
	Actor      Actor
}

// GitHubEventStatus is the internal domain status for a GitHubEvent decision.
// Serialised into the CR's status.contract field by the StatusWriter.
type GitHubEventStatus struct {
	// Accepted indicates the event passed validation and was admitted.
	Accepted bool
	// Triggered indicates the event caused downstream work to be dispatched.
	Triggered bool
	// Message is a human-readable explanation of the outcome.
	Message string
	// TriggeredRef is the name of the downstream resource created, if any.
	TriggeredRef string
}
