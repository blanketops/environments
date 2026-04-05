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

package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
)

// BackendSelector selects the backend responsible for handling GitHub events.
type BackendSelector struct {
	GitHub api.Provider
}

func NewBackendSelector(
	github api.Provider,
) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
	}
}

// ForEvent returns the provider that can handle this event.
// At this stage, all events are GitHub events by definition.
func (b *BackendSelector) ForEvent(_ domain.GitHubEvent) api.Provider {
	return b.GitHub
}

func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
