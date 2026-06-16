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
This file owns Result — the execution outcome produced after a BuildTrigger
is accepted and dispatched.

Result is distinct from Decision: Decision answers "should we execute?";
Result records what happened when we did. Result is produced by the execution
layer and carried back to the StatusWriter for persistence.
*/
package domain

import "time"

// Result is the outcome of a BuildTrigger execution cycle.
type Result struct {
	// Triggered indicates an execution was started.
	Triggered bool
	// Message is a human-readable summary of the outcome.
	Message string
	// ExecutionRef is the name of the object that was triggered
	// (e.g. BuildRun name).
	ExecutionRef string
	// ExecutionKind identifies what was triggered.
	// Examples: "Build", "BuildRun", "ExternalPipeline"
	ExecutionKind string
	// CorrelationID ties this result back to the originating external event
	// (e.g. GitHub delivery ID).
	CorrelationID string
	// At is when the result was produced.
	At time.Time
}
