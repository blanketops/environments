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

package serviceunit

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	bocache "github.com/BlanketOps/environments/cache"
	"github.com/BlanketOps/environments/core"
	serviceunitResolution "github.com/BlanketOps/environments/resolution/serviceunit"
)

// ServiceUnitCache provides domain-specific, field-level caching for ServiceUnit resources.
type ServiceUnitCache struct {
	*bocache.ObjectCache
}

// NewServiceUnitCache constructs a new ServiceUnitCache with the provided core.Cache.
func NewServiceUnitCache(c *core.Cache) *ServiceUnitCache {
	return &ServiceUnitCache{ObjectCache: bocache.NewObjectCache(c, "serviceunit", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: Type (static vs build)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetType(ctx context.Context, nn types.NamespacedName, gen int64, typ string) error {
	return s.SetField(ctx, nn, gen, "type", typ)
}

func (s *ServiceUnitCache) GetType(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var t string
	found, err := s.GetField(ctx, nn, gen, "type", &t)
	return t, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Image (for type: static)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetImage(ctx context.Context, nn types.NamespacedName, gen int64, image string) error {
	return s.SetField(ctx, nn, gen, "image", image)
}

func (s *ServiceUnitCache) GetImage(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var img string
	found, err := s.GetField(ctx, nn, gen, "image", &img)
	return img, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: BuildRef (for type: build)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, buildRef any) error {
	return s.SetField(ctx, nn, gen, "buildRef", buildRef)
}

func (s *ServiceUnitCache) GetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, into any) (bool, error) {
	return s.GetField(ctx, nn, gen, "buildRef", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: ContainerPort
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetContainerPort(ctx context.Context, nn types.NamespacedName, gen int64, port int) error {
	return s.SetField(ctx, nn, gen, "containerPort", port)
}

func (s *ServiceUnitCache) GetContainerPort(ctx context.Context, nn types.NamespacedName, gen int64) (int, bool, error) {
	var port int
	found, err := s.GetField(ctx, nn, gen, "containerPort", &port)
	return port, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Size
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetSize(ctx context.Context, nn types.NamespacedName, gen int64, size int) error {
	return s.SetField(ctx, nn, gen, "size", size)
}

func (s *ServiceUnitCache) GetSize(ctx context.Context, nn types.NamespacedName, gen int64) (int, bool, error) {
	var size int
	found, err := s.GetField(ctx, nn, gen, "size", &size)
	return size, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: AppType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetAppType(ctx context.Context, nn types.NamespacedName, gen int64, appType string) error {
	return s.SetField(ctx, nn, gen, "appType", appType)
}

func (s *ServiceUnitCache) GetAppType(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var at string
	found, err := s.GetField(ctx, nn, gen, "appType", &at)
	return at, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: StackType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetStackType(ctx context.Context, nn types.NamespacedName, gen int64, stackType string) error {
	return s.SetField(ctx, nn, gen, "stackType", stackType)
}

func (s *ServiceUnitCache) GetStackType(ctx context.Context, nn types.NamespacedName, gen int64) (string, bool, error) {
	var st string
	found, err := s.GetField(ctx, nn, gen, "stackType", &st)
	return st, found, err
}

// PublishResolved writes the resolved service unit contract as a
// generation-scoped, field-level projection. The image/buildRef pair is
// type-discriminated: exactly one is published per the unit's type. All
// writes are best-effort: failures cost queryability, never correctness.
// Returns the first error encountered for optional logging; callers
// should not fail reconciliation on it.
func (s *ServiceUnitCache) PublishResolved(ctx context.Context, nn types.NamespacedName, gen int64, r *serviceunitResolution.ResolvedServiceUnit) error {
	if r == nil || r.Spec == nil {
		return nil
	}
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	record(s.SetType(ctx, nn, gen, r.Spec.Type.String()))
	record(s.SetAppType(ctx, nn, gen, r.Spec.AppType))
	record(s.SetStackType(ctx, nn, gen, r.Spec.StackType))
	if r.Spec.ContainerPort > 0 {
		record(s.SetContainerPort(ctx, nn, gen, int(r.Spec.ContainerPort)))
	}
	if r.Spec.Size > 0 {
		record(s.SetSize(ctx, nn, gen, int(r.Spec.Size)))
	}
	// Type-discriminated: static units carry an image, build units a buildRef.
	switch r.Spec.Type.String() {
	case "static":
		record(s.SetImage(ctx, nn, gen, r.Spec.Image))
	case "build":
		if r.Spec.BuildRef != nil {
			record(s.SetBuildRef(ctx, nn, gen, r.Spec.BuildRef))
		}
	}
	return firstErr
}
