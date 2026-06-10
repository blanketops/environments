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

package cache

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

type Manager struct {
	client core.ExternalCache
}

// Get fetches a domain object, unmarshaling it into the provided pointer.
func (m *Manager) Get(ctx context.Context, item Cacheable, into any) (bool, error) {
	return m.client.Get(ctx, item.GetCacheKey(), into)
}

// Set stores the domain object.
func (m *Manager) Set(ctx context.Context, item Cacheable, val any) error {
	return m.client.Set(ctx, item.GetCacheKey(), val, item.GetTTL())
}
