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

	bocache "github.com/ntlaletsi70/blanketops-environments/cache"
	"github.com/ntlaletsi70/blanketops-environments/core"
)

// ServiceUnitCache provides domain-specific, field-level caching for ServiceUnit resources.
type ServiceUnitCache struct {
	*bocache.ObjectCache
}

// NewServiceUnitCache creates a new ServiceUnitCache instance.
func NewServiceUnitCache(c *core.Cache) *ServiceUnitCache {
	return &ServiceUnitCache{ObjectCache: bocache.NewObjectCache(c, "serviceunit", 0)}
}

// -----------------------------------------------------------------------------
// Typed Helpers: Type (static vs build)
// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------
// Typed Helpers: Image (for type: static)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetImage(ctx context.Context, nn types.NamespacedName, gen int64, name string, image string) error {
	return s.SetField(ctx, nn, gen, "image", image)
}

func (s *ServiceUnitCache) GetImage(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var img string
	found, err := s.GetField(ctx, nn, gen, "image", &img)
	return img, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: BuildRef (for type: build)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, name string, buildRef any) error {
	return s.SetField(ctx, nn, gen, "buildRef", buildRef)
}

func (s *ServiceUnitCache) GetBuildRef(ctx context.Context, nn types.NamespacedName, gen int64, name string, into any) (bool, error) {
	return s.GetField(ctx, nn, gen, "buildRef", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: ContainerPort
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetContainerPort(ctx context.Context, nn types.NamespacedName, gen int64, name string, port int) error {
	return s.SetField(ctx, nn, gen, "containerPort", port)
}

func (s *ServiceUnitCache) GetContainerPort(ctx context.Context, nn types.NamespacedName, gen int64, name string) (int, bool, error) {
	var port int
	found, err := s.GetField(ctx, nn, gen, "containerPort", &port)
	return port, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Size
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetSize(ctx context.Context, nn types.NamespacedName, gen int64, name string, size int) error {
	return s.SetField(ctx, nn, gen, "size", size)
}

func (s *ServiceUnitCache) GetSize(ctx context.Context, nn types.NamespacedName, gen int64, name string) (int, bool, error) {
	var size int
	found, err := s.GetField(ctx, nn, gen, "size", &size)
	return size, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: AppType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetAppType(ctx context.Context, nn types.NamespacedName, gen int64, name string, appType string) error {
	return s.SetField(ctx, nn, gen, "appType", appType)
}

func (s *ServiceUnitCache) GetAppType(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var at string
	found, err := s.GetField(ctx, nn, gen, "appType", &at)
	return at, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: StackType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetStackType(ctx context.Context, nn types.NamespacedName, gen int64, name string, stackType string) error {
	return s.SetField(ctx, nn, gen, "stackType", stackType)
}

func (s *ServiceUnitCache) GetStackType(ctx context.Context, nn types.NamespacedName, gen int64, name string) (string, bool, error) {
	var st string
	found, err := s.GetField(ctx, nn, gen, "stackType", &st)
	return st, found, err
}
