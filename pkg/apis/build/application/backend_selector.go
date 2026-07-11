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
This file owns BackendSelector — the routing layer that maps a resolved
BuildSpec to the correct build provider (Buildah, Kaniko, or Buildpacks).

Selection is driven by the StrategyName field using substring matching so
variant strategy names ("buildah-rootless", "kaniko-insecure") route correctly
without an exhaustive switch. Buildah is the fallback for unrecognised names.

BackendSelector sits in the application layer — it is called by the build
application service after resolution and before provider dispatch.
*/
package application

import (
	"strings"

	"github.com/BlanketOps/blankbtops-elanketops-environments/pkg/sp/uild/api"
domn
	"github.com/BlanketOps/environments/pkg/apis/build/domain"
)

// BackendSelector routes a BuildSpec to the correct Provider implementation
// based on the strategy name. All three providers must be non-nil at
// construction time.
type BackendSelector struct {
	Buildah    api.Provider
	Kaniko     api.Provider
	Buildpacks api.Provider
}

// NewBackendSelector constructs a BackendSelector with the three registered
// build providers.
func NewBackendSelector(buildah api.Provider, kaniko api.Provider, buildpacks api.Provider) *BackendSelector {
	return &BackendSelector{
		Buildah:    buildah,
		Kaniko:     kaniko,
		Buildpacks: buildpacks,
	}
}

// ForSpec returns the Provider that should execute the given BuildSpec.
// The first substring match against StrategyName wins. Buildah is returned
// for any unrecognised strategy name.
func (b *BackendSelector) ForSpec(spec domain.BuildSpec) api.Provider {
	switch {
	case strings.Contains(spec.StrategyName, "buildah"):
		return b.Buildah
	case strings.Contains(spec.StrategyName, "kaniko"):
		return b.Kaniko
	case strings.Contains(spec.StrategyName, "buildpacks"):
		return b.Buildpacks
	default:
		return b.Buildah
	}
}
