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

package events

import (
	"context"

	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"

	"github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

//
// ==============================
// DISPATCHER (ORCHESTRATION ONLY)
// ==============================
//

// GitHubDispatcher orchestrates:
// GitHub webhook → GitHubEvent → BuildTrigger → execution
//
// It contains NO business logic.
type GitHubDispatcher struct {
	GitHubAdapter   *githubevent.Adapter
	TriggerMatcher  BuildTriggerMatcher
	TriggerExecutor BuildTriggerMatcher
}

func NewGitHubDispatcher(
	githubAdapter *githubevent.Adapter,
	matcher BuildTriggerMatcher,
	executor BuildTriggerMatcher,
) *GitHubDispatcher {
	return &GitHubDispatcher{
		GitHubAdapter:   githubAdapter,
		TriggerMatcher:  matcher,
		TriggerExecutor: executor,
	}
}

//
// ==============================
// ENTRY POINT
// ==============================
//

func (d *GitHubDispatcher) Handle(
	ctx context.Context,
	event *eventsv1alpha1.GitHubEvent,
) error {

	// // 1. Resolve GitHubEvent (contract → runtime)
	// resolvedEvent, err := d.GitHubAdapter.Resolve(ctx, event)
	// if err != nil {
	// 	return err
	// }

	// // 2. Match BuildTriggers (USER INTENT)
	// triggers, err := d.TriggerMatcher.Match(ctx, resolvedEvent)
	// if err != nil {
	// 	return err
	// }

	// // 3. Execute triggers
	// for _, trigger := range triggers {
	// 	if err := d.TriggerExecutor.Execute(ctx, trigger, resolvedEvent); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}
