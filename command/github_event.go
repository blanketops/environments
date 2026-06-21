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
package command

import (
	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/core"
)

// TriggerOnGitHubDomain detects whether a command targets the GitHub event domain.
// Returns (repository, true) if the command is a GitHubEvent.
// Adjust the Spec field access to match your actual API shape.
func TriggerOnGitHubDomain(cmd core.Command) (string, bool) {
	if cmd.GVK.Kind != "GitHubEvent" {
		return "", false
	}

	_, ok := cmd.Obj.(*eventsv1alpha1.GitHubEvent)
	if !ok {
		return "", false
	}

	// TODO: adjust field name to match your GitHubEvent spec.
	// if gh.Spec.Repo != "" {
	// 	return gh.Spec.Repo, true
	// }

	return "", false
}
