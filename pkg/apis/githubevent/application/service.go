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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	githubeventResolution "github.com/BlanketOps/environments/resolution/githubevent"
	"github.com/BlanketOps/blanketops-environments/pkg/apis/githubevent/domain"
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
func (s *GitHubEventService) Reconcile(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent) error {
	// Map resolved contract → domain event.
	event := s.mapper.MapResolvedToDomain(resolved)

	// Select provider — currently always GitHub.
	provider := s.backend.Default()
	// Build conditions from outcome. This is the switch, moved from StatusWriter.

	// Provision or reconcile the Argo Events stack.
	result, err := provider.Ensure(ctx, resolved, event)
	conditions := s.buildConditions(result, err)

	// Write outcome to the GitHubEvent CR regardless of error.
	return s.status.Write(ctx, resolved.Event, conditions...)

}

// In your GitHubEventService (or a builder helper):
func (s *GitHubEventService) buildConditions(result domain.GitHubEventResult, runErr error) []metav1.Condition {
	now := metav1.NewTime(time.Now())

	switch {
	case runErr != nil:
		return []metav1.Condition{{
			Type:               "GitHubEventFailed",
			Status:             metav1.ConditionFalse,
			Reason:             "GitHubEventFailed",
			Message:            runErr.Error(),
			LastTransitionTime: now,
		}}
	case result.PayloadRecieved && result.Success:
		return []metav1.Condition{{
			Type:               "GitHubEventReady",
			Status:             metav1.ConditionTrue,
			Reason:             "GitHubEventPayloadReceived",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	case result.PayloadRecieved:
		return []metav1.Condition{{
			Type:               "GitHubEventReceiving",
			Status:             metav1.ConditionTrue,
			Reason:             "GitHubEventPayloadObserved",
			Message:            "Payload received, processing",
			LastTransitionTime: now,
		}}
	case result.Triggered:
		return []metav1.Condition{{
			Type:               "GitHubEventReady",
			Status:             metav1.ConditionTrue,
			Reason:             "GitHubEventReady",
			Message:            "GitHubEvent infrastructure provisioned",
			LastTransitionTime: now,
		}}
	default:
		return []metav1.Condition{{
			Type:               "GitHubEventPending",
			Status:             metav1.ConditionFalse,
			Reason:             "GitHubEventPending",
			Message:            result.Message,
			LastTransitionTime: now,
		}}
	}
}
