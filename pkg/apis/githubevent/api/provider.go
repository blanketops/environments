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
This file owns the Provider interface for the GitHubEvent domain — the contract
that all event ingress backends must satisfy.

A Provider is responsible for ensuring the observability infrastructure required
to receive and forward GitHub webhook deliveries exists and is healthy. The
concrete implementation (GitHubProvider in github.go) provisions the Argo
Events stack (EventBus, EventSource, Sensor).

Ensure is idempotent — calling it multiple times for the same GitHubEvent CR
must produce the same result with no unintended side effects.
*/
package api

import (
	"context"

	githubeventResolution "github.com/BlanketOps/environments/resolution/githubevent"
	"github.com/BlanketOps/environments/pkg/apis/githubevent/domain"
)

// Provider provisions and maintains the event ingress infrastructure for a
// GitHubEvent CR. Implementations must be idempotent.
type Provider interface {
	// Ensure creates or reconciles the infrastructure required to receive
	// webhook deliveries for the given GitHubEvent. Returns Accepted on
	// success and Rejected on failure.
	Ensure(
		ctx context.Context,
		resolved *githubeventResolution.ResolvedGitHubEvent,
		event domain.GitHubEvent,
	) (domain.GitHubEventResult, error)
}
