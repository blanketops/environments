package application

import (
	"context"

	pkgintent "github.com/ntlaletsi70/blanketops-environments/pkg/packages/intent"
	pkgResolution "github.com/ntlaletsi70/blanketops-environments/resolution/packages"
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
