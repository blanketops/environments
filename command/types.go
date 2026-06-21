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

package command

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// EventBus is the application-level CQRS event bus.
// Distinct from core.EventRecorder (Kubernetes events).
// Implementations live outside core; core never imports command.
type EventBus interface {
	Emit(ctx context.Context, event Event) error
}

// Event is the base interface for all CQRS domain events.
type Event interface {
	EventType() string
	EventTime() time.Time
}

// CommandReceived is emitted by the router as soon as a command enters the system.
// The observer writes "Pending" to the read model on this event.
type CommandReceived struct {
	CorrelationID string
	CommandType   core.CommandType
	GVK           schema.GroupVersionKind
	Name          string
	Namespace     string
	Timestamp     time.Time
}

func (e CommandReceived) EventType() string    { return "CommandReceived" }
func (e CommandReceived) EventTime() time.Time { return e.Timestamp }

// CommandRetrying is emitted when the router hits a conflict and will retry.
type CommandRetrying struct {
	CorrelationID string
	CommandType   core.CommandType
	GVK           schema.GroupVersionKind
	Attempt       int
	Reason        string
	Timestamp     time.Time
}

func (e CommandRetrying) EventType() string    { return "CommandRetrying" }
func (e CommandRetrying) EventTime() time.Time { return e.Timestamp }

// CommandSucceeded is emitted when engine execution succeeds.
type CommandSucceeded struct {
	CorrelationID string
	GVK           schema.GroupVersionKind
	Name          string
	Namespace     string
	Timestamp     time.Time
}

func (e CommandSucceeded) EventType() string    { return "CommandSucceeded" }
func (e CommandSucceeded) EventTime() time.Time { return e.Timestamp }

// CommandFailed is emitted when retries are exhausted or error is non-retryable.
type CommandFailed struct {
	CorrelationID string
	GVK           schema.GroupVersionKind
	Name          string
	Namespace     string
	Error         string
	Timestamp     time.Time
}

func (e CommandFailed) EventType() string    { return "CommandFailed" }
func (e CommandFailed) EventTime() time.Time { return e.Timestamp }

// ExternalGitHubTriggerEnqueued is emitted for commands targeting the GitHub domain.
type ExternalGitHubTriggerEnqueued struct {
	CorrelationID string
	Repository    string
	Timestamp     time.Time
}

func (e ExternalGitHubTriggerEnqueued) EventType() string    { return "ExternalGitHubTriggerEnqueued" }
func (e ExternalGitHubTriggerEnqueued) EventTime() time.Time { return e.Timestamp }
