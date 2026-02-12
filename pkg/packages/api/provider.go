package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/domain"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/intent"
)

// Provider executes a PackageIntent against a concrete backend (e.g. kapp).
type Provider interface {
	Execute(
		ctx context.Context,
		intent *intent.PackageIntent,
	) (*domain.PackageResult, error)
}
