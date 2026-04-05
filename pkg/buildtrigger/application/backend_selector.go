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

// pkg/buildtrigger/application/backend_selector.go
package application

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

// BackendSelector selects the provider responsible for evaluating BuildTriggers.
//
// IMPORTANT:
// - Selection is PURE
// - No side effects
// - No validation
// - No fallthrough logic hidden in providers
type BackendSelector struct {
	GitHub api.Provider
	Manual api.Provider
	// GitLab api.Provider (future)
}

// NewBackendSelector wires available providers.
func NewBackendSelector(
	github api.Provider,
	manual api.Provider,
) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
		Manual: manual,
	}
}

// ForTrigger returns the provider that can evaluate this trigger.
func (b *BackendSelector) ForTrigger(
	trigger domain.BuildTrigger,
) api.Provider {

	switch trigger.Trigger.Source {

	case domain.TriggerSourceGitHub:
		return b.GitHub

	case domain.TriggerSourceManual:
		return b.Manual

	default:
		// Defensive default — domain should already validate this
		return b.Default()
	}
}

// Default returns the fallback provider.
func (b *BackendSelector) Default() api.Provider {
	return b.GitHub
}
