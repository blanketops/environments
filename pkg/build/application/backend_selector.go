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
	"strings"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
)

type BackendSelector struct {
	Buildah    api.Provider
	Kaniko     api.Provider
	Buildpacks api.Provider
}

func NewBackendSelector(
	buildah api.Provider,
	kaniko api.Provider,
	buildpacks api.Provider,
) *BackendSelector {
	return &BackendSelector{
		Buildah:    buildah,
		Kaniko:     kaniko,
		Buildpacks: buildpacks,
	}
}

func (b *BackendSelector) ForSpec(spec domain.BuildSpec) api.Provider {
	name := spec.StrategyName

	switch {
	case strings.Contains(name, "buildah"):
		return b.Buildah

	case strings.Contains(name, "kaniko"):
		return b.Kaniko

	case strings.Contains(name, "buildpacks"):
		return b.Buildpacks

	default:
		// fallback
		return b.Buildah
	}
}
