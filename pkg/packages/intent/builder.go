package intent

import (
	"fmt"

	"github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/packages"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/domain"
)

// BuildPackageIntent compiles a RESOLVED Package into an immutable execution plan.
func BuildPackageIntent(
	rp *packages.ResolvedPackage,
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
