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
