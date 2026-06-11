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
<<<<<<< HEAD

import (
	"context"
	"fmt"
	"time"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// ServiceUnitCache provides domain-specific, field-level caching for ServiceUnit resources.
type ServiceUnitCache struct {
	cache *core.Cache
}

// NewServiceUnitCache creates a new ServiceUnitCache instance.
func NewServiceUnitCache(c *core.Cache) *ServiceUnitCache {
	return &ServiceUnitCache{cache: c}
}

// key generates a specific cache key for a single field of a ServiceUnit.
func (s *ServiceUnitCache) key(name, field string) string {
	return fmt.Sprintf("serviceunit:%s:%s", name, field)
}

// SetField stores an individual CR value in the external cache.
func (s *ServiceUnitCache) SetField(ctx context.Context, name, field string, val any) error {
	return s.cache.External.Set(ctx, s.key(name, field), val, 1*time.Hour)
}

// GetField retrieves an individual CR value from the external cache.
func (s *ServiceUnitCache) GetField(ctx context.Context, name, field string, into any) (bool, error) {
	return s.cache.External.Get(ctx, s.key(name, field), into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: Type (static vs build)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetType(ctx context.Context, name string, unitType string) error {
	return s.SetField(ctx, name, "type", unitType)
}

func (s *ServiceUnitCache) GetType(ctx context.Context, name string) (string, bool, error) {
	var t string
	found, err := s.GetField(ctx, name, "type", &t)
	return t, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Image (for type: static)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetImage(ctx context.Context, name string, image string) error {
	return s.SetField(ctx, name, "image", image)
}

func (s *ServiceUnitCache) GetImage(ctx context.Context, name string) (string, bool, error) {
	var img string
	found, err := s.GetField(ctx, name, "image", &img)
	return img, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: BuildRef (for type: build)
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetBuildRef(ctx context.Context, name string, buildRef any) error {
	return s.SetField(ctx, name, "buildRef", buildRef)
}

func (s *ServiceUnitCache) GetBuildRef(ctx context.Context, name string, into any) (bool, error) {
	return s.GetField(ctx, name, "buildRef", into)
}

// -----------------------------------------------------------------------------
// Typed Helpers: ContainerPort
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetContainerPort(ctx context.Context, name string, port int) error {
	return s.SetField(ctx, name, "containerPort", port)
}

func (s *ServiceUnitCache) GetContainerPort(ctx context.Context, name string) (int, bool, error) {
	var port int
	found, err := s.GetField(ctx, name, "containerPort", &port)
	return port, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: Size
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetSize(ctx context.Context, name string, size int) error {
	return s.SetField(ctx, name, "size", size)
}

func (s *ServiceUnitCache) GetSize(ctx context.Context, name string) (int, bool, error) {
	var size int
	found, err := s.GetField(ctx, name, "size", &size)
	return size, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: AppType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetAppType(ctx context.Context, name string, appType string) error {
	return s.SetField(ctx, name, "appType", appType)
}

func (s *ServiceUnitCache) GetAppType(ctx context.Context, name string) (string, bool, error) {
	var at string
	found, err := s.GetField(ctx, name, "appType", &at)
	return at, found, err
}

// -----------------------------------------------------------------------------
// Typed Helpers: StackType
// -----------------------------------------------------------------------------

func (s *ServiceUnitCache) SetStackType(ctx context.Context, name string, stackType string) error {
	return s.SetField(ctx, name, "stackType", stackType)
}

func (s *ServiceUnitCache) GetStackType(ctx context.Context, name string) (string, bool, error) {
	var st string
	found, err := s.GetField(ctx, name, "stackType", &st)
	return st, found, err
}
=======
>>>>>>> main
