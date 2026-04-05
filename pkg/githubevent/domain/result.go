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

type GitHubEventResult struct {
	Accepted     bool
	Triggered    bool
	Message      string
	TriggeredRef string // optional: Build name, Deployment, etc
}

// ToStatus converts a GitHubEventResult to a GitHubEventStatus.
func (r GitHubEventResult) ToStatus() GitHubEventStatus {
	return GitHubEventStatus(r)
}

func Accepted(msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted: true,
		Message:  msg,
	}
}

func Triggered(ref, msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted:     true,
		Triggered:    true,
		TriggeredRef: ref,
		Message:      msg,
	}
}

func Rejected(msg string) GitHubEventResult {
	return GitHubEventResult{
		Accepted: false,
		Message:  msg,
	}
}
