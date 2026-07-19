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
	"fmt"

	"github.com/blanketops/environments/pkg/apis/packages/domain"
	"github.com/blanketops/environments/resolution/packages/resolve"
)

// BuildPackageIntent compiles a RESOLVED Package into an immutable execution plan.
func BuildPackageIntent(
	rp *resolve.ResolvedPackage,
) (*PackageIntent, error) {

	if rp == nil || rp.Spec == nil {
		return nil, fmt.Errorf("nil ResolvedPackage passed to BuildPackageIntent")
	}

	spec := rp.Spec

	if !spec.Enabled {
		return nil, fmt.Errorf("package %s is disabled", rp.Package.Name)
	}

	// ------------------------------------------------------------
	// Identity (already normalized)
	// ------------------------------------------------------------
	id := domain.PackageID{
		Name:      rp.Package.Name,
		Namespace: rp.Package.Namespace,
	}

	// ------------------------------------------------------------
	// Source (already validated by resolver)
	// ------------------------------------------------------------
	source := domain.PackageSource{
		RepositoryURL:     spec.PackageRepository.URL,
		CredentialsSecret: spec.PackageRepository.CredentialsSecret,
	}

	// ------------------------------------------------------------
	// State repository (already validated)
	// ------------------------------------------------------------
	stateRepo := domain.StateRepository{
		URL:         spec.StateRepository.URL,
		Path:        spec.StateRepository.Path,
		Strategy:    spec.StateRepository.Strategy,
		CloneSecret: spec.StateRepository.CloneSecret,
		Ref: domain.Ref{
			Branch: spec.StateRepository.Ref.Branch,
			Tag:    spec.StateRepository.Ref.Tag,
			Commit: spec.StateRepository.Ref.Commit,
		},
	}

	// ------------------------------------------------------------
	// Execution behavior (resolver-owned semantics)
	// ------------------------------------------------------------
	intent := &PackageIntent{
		ID:          id,
		Source:      source,
		StateRepo:   stateRepo,
		DiffEnabled: spec.DiffEnabled,
		Strategy:    domain.ApplyStrategy(spec.StateRepository.Strategy),
		//ResolvedRef:    spec.ResolvedRef,
		//ResolvedCommit: spec.ResolvedCommit,
	}

	return intent, nil
}
