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

package environment

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/blanketops/environments/cache"
	"github.com/blanketops/environments/core/cache"
	environmentResolution "github.com/blanketops/environments/resolution/environment"
)

// EnvironmentCache provides domain-specific, field-level caching for Environment resources.
type EnvironmentCache struct {
	*bocache.ObjectCache
}

// NewEnvironmentCache constructs a new EnvironmentCache with the provided cache.Cache.
func NewEnvironmentCache(c *cache.Cache) *EnvironmentCache {
	return &EnvironmentCache{ObjectCache: bocache.NewObjectCache(c, "environment", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: ApplicationName
// -----------------------------------------------------------------------------

func (e *EnvironmentCache) SetApplicationName(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "applicationName", name)
}

func (e *EnvironmentCache) GetApplicationName(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "applicationName", &v)
	return v, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: EnvironmentType
// -----------------------------------------------------------------------------

func (e *EnvironmentCache) SetEnvironmentType(ctx context.Context, nn types.NamespacedName, gen int64, envType string) error {
	return e.SetField(ctx, nn, gen, "environmentType", envType)
}

func (e *EnvironmentCache) GetEnvironmentType(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "environmentType", &v)
	return v, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Version
// -----------------------------------------------------------------------------

func (e *EnvironmentCache) SetVersion(ctx context.Context, nn types.NamespacedName, gen int64, version string) error {
	return e.SetField(ctx, nn, gen, "version", version)
}

func (e *EnvironmentCache) GetVersion(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "version", &v)
	return v, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: SecretStoreProvider
// -----------------------------------------------------------------------------

func (e *EnvironmentCache) SetSecretStoreProvider(ctx context.Context, nn types.NamespacedName, gen int64, provider string) error {
	return e.SetField(ctx, nn, gen, "secretStoreProvider", provider)
}

func (e *EnvironmentCache) GetSecretStoreProvider(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "secretStoreProvider", &v)
	return v, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: CR refs
// -----------------------------------------------------------------------------

func (e *EnvironmentCache) SetBuild(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "build", name)
}

func (e *EnvironmentCache) GetBuild(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "build", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetGitRepository(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "gitRepository", name)
}

func (e *EnvironmentCache) GetGitRepository(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "gitRepository", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetGitHubEvent(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "gitHubEvent", name)
}

func (e *EnvironmentCache) GetGitHubEvent(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "gitHubEvent", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetDeployment(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "deployment", name)
}

func (e *EnvironmentCache) GetDeployment(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "deployment", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetRoute(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "route", name)
}

func (e *EnvironmentCache) GetRoute(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "route", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetPackage(ctx context.Context, nn types.NamespacedName, gen int64, name string) error {
	return e.SetField(ctx, nn, gen, "package", name)
}

func (e *EnvironmentCache) GetPackage(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var v string
	found, err := e.GetField(ctx, nn, gen, "package", &v)
	return v, found, err
}

func (e *EnvironmentCache) SetServiceUnits(ctx context.Context, nn types.NamespacedName, gen int64, units []string) error {
	return e.SetField(ctx, nn, gen, "serviceUnits", units)
}

func (e *EnvironmentCache) GetServiceUnits(ctx context.Context, nn types.NamespacedName, gen int64) ([]string, bool, error) {
	var v []string
	found, err := e.GetField(ctx, nn, gen, "serviceUnits", &v)
	return v, found, err
}

// -----------------------------------------------------------------------------
// PublishResolved
// -----------------------------------------------------------------------------

// PublishResolved writes the resolved environment contract as a
// generation-scoped, field-level projection. All writes are best-effort:
// failures cost queryability, never correctness. Returns the first error
// encountered for optional logging; callers must not fail reconciliation on it.
func (e *EnvironmentCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *environmentResolution.ResolvedEnvironment) error {
	if r == nil || r.Spec == nil {
		return nil
	}

	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Mandatory: present whenever resolution succeeds.
	record(e.SetApplicationName(ctx, nn, gen, r.Spec.ApplicationName))
	record(e.SetEnvironmentType(ctx, nn, gen, r.Spec.EnvironmentType))
	record(e.SetVersion(ctx, nn, gen, r.Spec.Version))

	// Conditional: publish only when resolved to something meaningful.
	if r.Spec.Contract != nil && r.Spec.Contract.SecretStoreProvider != "" {
		record(e.SetSecretStoreProvider(ctx, nn, gen, r.Spec.Contract.SecretStoreProvider))
	}
	if r.Spec.Build != "" {
		record(e.SetBuild(ctx, nn, gen, r.Spec.Build))
	}
	if r.Spec.GitRepository != "" {
		record(e.SetGitRepository(ctx, nn, gen, r.Spec.GitRepository))
	}
	if r.Spec.GitHubEvent != "" {
		record(e.SetGitHubEvent(ctx, nn, gen, r.Spec.GitHubEvent))
	}
	if r.Spec.Deployment != "" {
		record(e.SetDeployment(ctx, nn, gen, r.Spec.Deployment))
	}
	if r.Spec.Route != "" {
		record(e.SetRoute(ctx, nn, gen, r.Spec.Route))
	}
	if r.Spec.Package != "" {
		record(e.SetPackage(ctx, nn, gen, r.Spec.Package))
	}
	if len(r.Spec.ServiceUnits) > 0 {
		record(e.SetServiceUnits(ctx, nn, gen, r.Spec.ServiceUnits))
	}

	return firstErr
}
