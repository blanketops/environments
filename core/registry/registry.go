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
Package registry owns the Registry — the runtime lookup table that wires
Domains and strategies into the Engine. The Registry is populated once at
manager startup and then read concurrently by the Engine during
reconciliation. All methods are safe for concurrent use.

Two registries are maintained in a single struct:

  - Domain registry: maps GVK → Domain. The Engine uses this to route every
    Command to the correct handler. One Domain per GVK — duplicate registration
    silently overwrites, so startup order matters.

  - Strategy registry: maps name → implementation (any). Used for pluggable
    build strategies (e.g. "buildpacks-v3", "dockerfile"). Typed retrieval
    is the caller's responsibility via a type assertion after GetStrategy.
*/
package registry

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

	domain "github.com/blanketops/environments/core/domain"
)

// Registry is the runtime lookup table for Domains and strategies.
// Populated at manager startup; read concurrently by the Engine thereafter.
// All methods acquire the appropriate lock — callers need no external
// synchronisation.
type Registry struct {
	// mu guards both domains and strategies maps.
	mu sync.RWMutex

	// domains maps GroupVersionKind → Domain for Engine routing.
	// One Domain per GVK; duplicate registration overwrites silently.
	domains map[schema.GroupVersionKind]domain.Domain

	// strategies maps strategy name → implementation for pluggable
	// build strategy dispatch. Typed access requires a caller-side assertion.
	strategies map[string]any
}

// NewRegistry constructs an empty Registry ready for domain and strategy
// registration. Called once during manager setup before any controllers start.
func NewRegistry() *Registry {
	return &Registry{
		domains:    make(map[schema.GroupVersionKind]domain.Domain),
		strategies: make(map[string]any),
	}
}

// RegisterDomain registers a Domain for the given GVK. Called at startup for
// each CR kind the platform manages. Duplicate registration overwrites the
// previous entry — registration order is the caller's responsibility.
func (r *Registry) RegisterDomain(gvk schema.GroupVersionKind, d domain.Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[gvk] = d
}

// GetDomain retrieves the Domain registered for gvk. Returns false if no
// Domain is registered — the Engine surfaces this as an error to the caller.
func (r *Registry) GetDomain(gvk schema.GroupVersionKind) (domain.Domain, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.domains[gvk]
	return d, ok
}

// RegisterStrategy registers a named strategy implementation. The name is
// the lookup key used at reconciliation time (e.g. "buildpacks-v3",
// "dockerfile"). Duplicate names overwrite silently.
func (r *Registry) RegisterStrategy(name string, v any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[name] = v
}

// GetStrategy retrieves a strategy by name. The caller is responsible for
// type-asserting the returned value to the concrete strategy interface.
// Returns false if no strategy is registered under name.
func (r *Registry) GetStrategy(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.strategies[name]
	return v, ok
}

// ListDomains returns the GVKs of all registered Domains. Order is
// non-deterministic (map iteration). Intended for startup logging and
// diagnostics only — not for routing decisions.
func (r *Registry) ListDomains() []schema.GroupVersionKind {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]schema.GroupVersionKind, 0, len(r.domains))
	for k := range r.domains {
		out = append(out, k)
	}
	return out
}

// ListStrategies returns the names of all registered strategies. Order is
// non-deterministic (map iteration). Intended for startup logging and
// diagnostics only.
func (r *Registry) ListStrategies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.strategies))
	for k := range r.strategies {
		out = append(out, k)
	}
	return out
}
