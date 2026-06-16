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
This file owns GitHubEventService — the application service that orchestrates
GitHubEvent reconciliation.

The pipeline mirrors the build and buildtrigger domains: map, select, execute,
write. The service does not provision Argo resources directly — that belongs to
the Provider. It does not make admission decisions — those belong to the
provider result. It only sequences the four steps and ensures the CR always
reflects the latest outcome.
*/
package application

import (
	"context"

	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// GitHubEventService orchestrates GitHubEvent reconciliation.
// Stateless and safe for concurrent use.
type GitHubEventService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

// NewGitHubEventService constructs a GitHubEventService with the required
// collaborators.
func NewGitHubEventService(
	mapper *Mapper,
	status *StatusWriter,
	backend *BackendSelector,
) *GitHubEventService {
	return &GitHubEventService{mapper: mapper, status: status, backend: backend}
}

// Reconcile maps the resolved event to a domain model, delegates provisioning
// to the provider, then writes the result back to the GitHubEvent CR.
// Status is always written — even when provisioning fails.
func (s *GitHubEventService) Reconcile(
	ctx context.Context,
	resolved *githubeventResolution.ResolvedGitHubEvent,
) error {
	// Map resolved contract → domain event.
	event := s.mapper.MapResolvedToDomain(resolved)

	// Select provider — currently always GitHub.
	provider := s.backend.Default()

	// Provision or reconcile the Argo Events stack.
	result, err := provider.Ensure(ctx, resolved, event)

	// Write outcome to the GitHubEvent CR regardless of error.
	return s.status.Write(ctx, resolved.Event, result, err)
}
