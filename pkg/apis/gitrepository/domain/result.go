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

// Result represents the outcome of attempting to realize
// a GitRepository registration.
type Result struct {
	State  State
	Reason string
}

// Success returns a successful realization result.
func Success() Result {
	return Result{
		State:  StateReady,
		Reason: "repository and webhooks realized",
	}
}

// Pending returns a pending realization result.
func Pending(reason string) Result {
	return Result{
		State:  StatePending,
		Reason: reason,
	}
}

// Failure returns a failed realization result.
func Failure(reason string) Result {
	return Result{
		State:  StateError,
		Reason: reason,
	}
}
