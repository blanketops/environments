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
Package domain — adapter.go

This file owns the Adapter — a thin struct wrapper around the package-level
ResolveDomain function. The Adapter satisfies interface-based injection points
in the application layer where a concrete resolution dependency must be passed
as a value rather than called as a free function.

Stateless by design — all resolution logic lives in resolve.go. Future deps
(client, logger) will be wired here when cross-CR reads or observability hooks
are required. Until then, resolution is pure and nothing is needed at
construction time.

See also:
  - resolution/domain/resolve.go          — resolution entry point and types
  - resolution/domain/contract_adapter.go — one-way proto projection
  - pkg/domain/application/service.go     — primary consumer of Adapter.Resolve
*/
package domain

import (
	"context"

	networksv1alpha1 "github.com/BlanketOps/environments-api/api/networks/v1alpha1"
)

// Adapter wraps the Domain resolution function for interface-based injection.
// Stateless — all resolution logic lives in ResolveDomain.
//
// Planned future deps (add when needed):
//   - client client.Client — for cross-CR reads (e.g. verifying Route ownership)
//   - log    logr.Logger   — for resolution observability
type Adapter struct {
	// future deps:
	// client client.Client
	// log    logr.Logger
}

// NewAdapter constructs a stateless Domain resolution Adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Resolve delegates to ResolveDomain, returning the validated typed contract
// or an error if spec.contract fails validation.
func (a *Adapter) Resolve(ctx context.Context, domain *networksv1alpha1.Domain) (*ResolvedDomain, error) {
	return ResolveDomain(domain)
}
