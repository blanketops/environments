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

// State represents the lifecycle state of a GitRepository registration.
type State string

const (
	// StatePending means the repository has been declared
	// but has not yet been realized.
	StatePending State = "Pending"

	// StateReady means the repository and all required webhooks
	// are realized and match desired intent.
	StateReady State = "Ready"

	// StateError means realization failed and requires attention.
	StateError State = "Error"
)

// IsTerminal indicates whether the state is terminal.
func (s State) IsTerminal() bool {
	return s == StateReady || s == StateError
}
