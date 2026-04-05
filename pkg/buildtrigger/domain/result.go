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

import "time"

// Result represents the outcome produced by a BuildTrigger.
type Result struct {
	// Whether the trigger resulted in an execution being started
	Triggered bool

	// Human-readable summary of the outcome
	Message string

	// Reference to the execution that was triggered
	// (e.g. Build name, BuildRun name, or external ref)
	ExecutionRef string

	// ExecutionKind identifies what was triggered
	// Examples: "Build", "BuildRun", "ExternalPipeline"
	ExecutionKind string

	// CorrelationID ties this trigger result back to an external event
	// (e.g. GitHub delivery ID, webhook UUID)
	CorrelationID string

	// When the result was produced
	At time.Time
}
