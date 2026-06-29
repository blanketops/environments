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

package intent

import (
	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/packages/domain"
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
