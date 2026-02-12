package application

import (
	packageapi "github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/api"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/packages/intent"
)

// BackendSelector is intentionally boring.
// Packages are ALWAYS executed via kapp.
// No strategy selection is allowed.
type BackendSelector struct {
	Kapp packageapi.Provider
}

// NewBackendSelector constructs a fixed kapp executor.
func NewBackendSelector(
	kapp packageapi.Provider,
) *BackendSelector {
	return &BackendSelector{
		Kapp: kapp,
	}
}

// ForIntent always returns kapp.
// Intent does NOT influence execution backend.
func (b *BackendSelector) ForIntent(
	_ *intent.PackageIntent,
) packageapi.Provider {

	if b.Kapp == nil {
		panic("kapp provider must be configured for PackageService")
	}

	return b.Kapp
}
