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
This file owns the State type — the lifecycle state of a GitHubEvent CR as
tracked by the domain layer.

State is intentionally coarse: the domain only cares whether an event is new,
has been handled, was deliberately ignored, or errored. Fine-grained progress
is tracked by the provider and surfaced via conditions.
*/
package domain

// State is the lifecycle state of a GitHubEvent.
type State string

const (
	// StateNew — event received, not yet evaluated.
	StateNew State = "New"
	// StateHandled — event was admitted and downstream work was dispatched.
	StateHandled State = "Handled"
	// StateIgnored — event was admitted but no action was required.
	StateIgnored State = "Ignored"
	// StateError — evaluation failed due to a provider or infrastructure error.
	StateError State = "Error"
)
