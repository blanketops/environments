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
Package intent (imported as pkg/intent/package — "package" is a Go
keyword, so the directory is named singular while the package itself is
still named intent, matching pkg/intent/deployment and
pkg/intent/serviceunit's naming) owns PackageIntent, the compiled,
immutable execution plan for a Package CR: its stable identity, manifest
source, GitOps state repository, and apply strategy.

BuildPackageIntent (builder.go) is the constructor, compiling a resolved
Package (resolution/packages/resolve.ResolvedPackage) into a PackageIntent
once ahead of execution — the same "resolve once, execute from an
immutable plan" shape used by pkg/intent/deployment.IntentBuilder.
*/
package intent

import (
	"github.com/blanketops/environments/pkg/apis/packages/domain"
)

// PackageIntent is the compiled, immutable execution plan.
type PackageIntent struct {
	// Stable identity
	ID domain.PackageID

	// Source of manifests
	Source domain.PackageSource

	// State repository (GitOps anchor)
	StateRepo domain.StateRepository

	// Execution behavior
	DiffEnabled bool
	Strategy    domain.ApplyStrategy

	// Resolved ref (branch/tag/commit)
	ResolvedRef    string
	ResolvedCommit string
}
