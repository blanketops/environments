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
Package build implements resolution for the Build CR.

This file owns the Adapter — a thin struct wrapper around the package-level
ResolveBuild function. The Adapter exists to satisfy interface-based injection
points in the application layer where a concrete resolution dependency must be
passed as a value rather than called as a free function.

Current deps are intentionally empty — client and logger will be added here
when resolution requires cross-CR reads or observability hooks. Until then,
resolution is pure and stateless so no deps are needed.
*/
package build

import (
	"context"

	environmentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

// Adapter wraps the Build resolution function for interface-based injection.
// Stateless — all resolution logic lives in ResolveBuild.
//
// Planned future deps (add when needed):
//   - client client.Client — for cross-CR reads during resolution.
//   - log    logr.Logger   — for resolution observability.
type Adapter struct {
	// future deps:
	// client client.Client
	// log    logr.Logger
}

// NewAdapter constructs a stateless Build resolution Adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Resolve delegates to ResolveBuild, returning the resolved Build contract
// or an error if the CR spec fails validation.
func (a *Adapter) Resolve(
	ctx context.Context,
	build *environmentv1alpha1.Build,
) (*ResolvedBuild, error) {
	return ResolveBuild(build)
}
