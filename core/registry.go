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

package core

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Registry holds runtime registrations for domains and optional strategies.
type Registry struct {
	mu         sync.RWMutex
	domains    map[schema.GroupVersionKind]Domain
	strategies map[string]any
}

// NewRegistry constructs an empty runtime registry.
func NewRegistry() *Registry {
	return &Registry{
		domains:    make(map[schema.GroupVersionKind]Domain),
		strategies: make(map[string]any),
	}
}

// RegisterDomain registers a domain in the runtime registry.
func (r *Registry) RegisterDomain(gvk schema.GroupVersionKind, d Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[gvk] = d
}

// GetDomain retrieves a domain by its GroupVersionKind.
func (r *Registry) GetDomain(gvk schema.GroupVersionKind) (Domain, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.domains[gvk]
	return d, ok
}

// RegisterStrategy registers a named strategy implementation (e.g., "buildpacks-v3").
func (r *Registry) RegisterStrategy(name string, v any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[name] = v
}

// GetStrategy retrieves a strategy by name.
func (r *Registry) GetStrategy(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.strategies[name]
	return v, ok
}

// ListDomains returns a slice of registered GVKs (for debugging/logging).
func (r *Registry) ListDomains() []schema.GroupVersionKind {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]schema.GroupVersionKind, 0, len(r.domains))
	for k := range r.domains {
		out = append(out, k)
	}
	return out
}

// ListStrategies returns all registered strategy names (for debugging/logging).
func (r *Registry) ListStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.strategies))
	for k := range r.strategies {
		out = append(out, k)
	}
	return out
}
