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
This file owns TriggerContext — the execution identity metadata extracted from
Build annotations by the provider layer.

TriggerContext is combined with the resolved BuildSpec to produce the
deterministic execution hash used for BuildRun deduplication. It is populated
by ExtractTriggerContext (pkg/build/api/trigger_context.go) and consumed only
by utils.ComputeExecutionHash — it does not flow into the domain model or the
resolved spec.
*/
package api

// TriggerContext carries the trigger metadata written to Build annotations by
// the BuildTrigger domain when it patches the Build CR before dispatching.
// Absent annotations default to manual trigger with no retry context.
type TriggerContext struct {
	// Type is the event kind that caused this execution.
	// One of: manual | push | pull_request | schedule
	Type string
	// Ref is the Git ref the trigger applies to.
	// e.g. refs/heads/main, refs/pull/123/head
	Ref string
	// SHA is the commit SHA carried by the trigger event.
	SHA string
	// RetryAttempt identifies the current retry cycle.
	// Values: on-failure | always | never
	RetryAttempt string
}
