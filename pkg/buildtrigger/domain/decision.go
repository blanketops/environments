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
This file owns Decision — the pure output of trigger policy evaluation.

Decision is returned by the Provider interface and consumed by the application
service to determine whether to proceed with execution. It carries no side
effects and does not interact with Kubernetes.
*/
package domain

// Decision is the result of evaluating a BuildTrigger against policy.
// Returned by Provider.Evaluate — pure, no side effects.
type Decision struct {
	// Accepted indicates the trigger passed policy evaluation.
	Accepted bool
	// Execute indicates the trigger should proceed to BuildRun dispatch.
	// A trigger may be accepted but not executed (e.g. dry-run mode).
	Execute bool
	// Message is a human-readable explanation of the decision.
	Message string
	// ExecutionRef is the name of the execution object if one was pre-allocated.
	// Optional — populated by providers that allocate the run before returning.
	ExecutionRef string
}
