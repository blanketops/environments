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

// PackageResult represents the outcome of a single reconciliation run.
type PackageResult struct {
	// ---------------------------------------------------------------------
	// Outcome
	// ---------------------------------------------------------------------

	Success bool
	Phase   PackagePhase
	Message string

	// ---------------------------------------------------------------------
	// Source-of-truth resolution
	// ---------------------------------------------------------------------

	// DesiredRef is what the controller intended to apply.
	DesiredRef string

	// DesiredCommit is the resolved commit hash.
	DesiredCommit string

	// ---------------------------------------------------------------------
	// kapp execution details
	// ---------------------------------------------------------------------

	Kapp KappResult

	// ---------------------------------------------------------------------
	// Timing
	// ---------------------------------------------------------------------

	StartedAt  time.Time
	FinishedAt time.Time
}

// -----------------------------------------------------------------------------
// kapp execution result
// -----------------------------------------------------------------------------

// KappResult captures the outcome of a single kapp invocation during
// package reconciliation.
type KappResult struct {
	// Executed indicates whether kapp was invoked at all.
	Name      string
	Namespace string
	Executed  bool

	// Deploy timing (authoritative)
	DeployStartedAt *time.Time
	DeployUpdatedAt *time.Time

	// Exit info
	DeployFinished bool
	DeployExitCode *int

	// Action taken by kapp.
	// e.g. diff, apply, noop
	Action KappAction

	// Diff indicates whether differences were detected.
	Diff DiffState

	// Output is raw kapp stdout/stderr (trimmed upstream if needed).
	Output string

	// Error captures execution failure, if any.
	Error string
}

// -----------------------------------------------------------------------------
// kapp actions
// -----------------------------------------------------------------------------

// KappAction identifies which operation kapp performed (or would perform).
type KappAction string

const (
	KappActionNone  KappAction = "None"
	KappActionDiff  KappAction = "Diff"
	KappActionApply KappAction = "Apply"
)
