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
This file owns BackendSelector for the GitHubEvent domain. At present there is
only one provider (GitHub), so selection is always a pass-through. The selector
exists to preserve the same layering pattern as the build and buildtrigger
domains and to make adding future providers (GitLab, Bitbucket) a localised
change.
*/
package application

import (
	"github.com/BlanketOps/environments/pkg/apis/githubevent/api"
	"github.com/BlanketOps/environments/pkg/apis/githubevent/domain"
)

// BackendSelector routes a GitHubEvent to the correct Provider.
// Currently always returns the GitHub provider.
type BackendSelector struct {
	GitHub api.Provider
}

// NewBackendSelector constructs a BackendSelector with the GitHub provider.
func NewBackendSelector(github api.Provider) *BackendSelector {
	return &BackendSelector{GitHub: github}
}

// ForEvent returns the provider for the given event. All events are currently
// GitHub events so selection is unconditional.
func (b *BackendSelector) ForEvent(_ domain.GitHubEvent) api.Provider {
	return b.GitHub
}

// Default returns the GitHub provider — the only registered provider.
func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
