package api

import (
	"context"

	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/gitrepository/domain"
)

// Provider defines a backend capable of realizing a GitRepository declaration.
type Provider interface {
	// Ensure accepts context, the GitRepository CR (for metadata / owner refs),
	// and the pure domain model representing intent.
	Ensure(
		ctx context.Context,
		cr *sourcesv1alpha1.GitRepository,
		spec domain.GitRepository,
	) (domain.Result, error)
}
