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

package application

import (
	"context"

	pkgintent "github.com/blanketops/environments/pkg/intent/package"
	pkgResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

type PackageService struct {
	mapper  *Mapper
	backend *BackendSelector
	status  *StatusWriter
}

func NewPackageService(
	mapper *Mapper,
	backend *BackendSelector,
	status *StatusWriter,
) *PackageService {
	return &PackageService{
		mapper:  mapper,
		backend: backend,
		status:  status,
	}
}

// Reconcile executes a PackageIntent and writes authoritative status.
func (s *PackageService) Reconcile(
	ctx context.Context,
	resolved *pkgResolution.ResolvedPackage,
	intent *pkgintent.PackageIntent,
) error {

	// ------------------------------------------------
	// 1. Select backend (always kapp)
	// ------------------------------------------------
	provider := s.backend.ForIntent(intent)

	// ------------------------------------------------
	// 2. Execute package (INTENT → RESULT)
	// ------------------------------------------------
	result, err := provider.Execute(ctx, intent)

	// ------------------------------------------------
	// 3. Write status (AUTHORITATIVE CR)
	// ------------------------------------------------
	return s.status.Write(
		ctx,
		resolved.Package, // ✅ authoritative *envv1.Package
		result,
		err,
	)
}
