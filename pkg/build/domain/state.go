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

// IsTerminal reports whether the build execution has reached a terminal state.
//
// A build is terminal once an execution has been triggered and the controller
// has recorded an outcome (success or failure).
//
// Progress and intermediate states are tracked by observers (e.g. BuildRun),
// not by the domain model.
func (s BuildStatus) IsTerminal() bool {
	if !s.Triggered {
		return false
	}

	// Once triggered, the controller records exactly one outcome.
	// Success=true  -> terminal (succeeded)
	// Success=false -> terminal (failed)
	return true
}
