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
This file owns BackendSelector for the BuildTrigger domain — the routing layer
that maps a resolved trigger source to the correct evaluation provider.

Selection is pure: no side effects, no validation, no policy. The domain layer
is responsible for validating the trigger source before ForTrigger is called.
GitHub is the fallback for unrecognised sources.
*/
package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

// BackendSelector routes a BuildTrigger to the Provider responsible for
// evaluating it. Selection is driven by the trigger source — one provider
// per source kind.
type BackendSelector struct {
	GitHub api.Provider
	Manual api.Provider
	// GitLab api.Provider — future
}

// NewBackendSelector constructs a BackendSelector with the registered providers.
func NewBackendSelector(github api.Provider, manual api.Provider) *BackendSelector {
	return &BackendSelector{GitHub: github, Manual: manual}
}

// ForTrigger returns the Provider that should evaluate the given BuildTrigger.
// Falls back to GitHub for unrecognised sources — the domain layer validates
// the source before this is called.
func (b *BackendSelector) ForTrigger(trigger domain.BuildTrigger) api.Provider {
	switch trigger.Trigger.Source {
	case domain.TriggerSourceGitHub:
		return b.GitHub
	case domain.TriggerSourceManual:
		return b.Manual
	default:
		return b.Default()
	}
}

// Default returns the fallback provider (GitHub).
func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
