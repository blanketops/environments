package intent

import (
	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"

	"time"
)

type DeploymentIntent struct {
	Source *environmentv1.Deployment // ← IMPORTANT

	Name      string
	Namespace string

	Runtime  Runtime
	Strategy Strategy

	ServiceUnits []ServiceUnitIntent

	ReconciliationStrategy ReconciliationStrategy
	ImageAutomation        bool
	ManifestsRepo          *ManifestsRepo

	GitOwner    string
	GeneratedAt time.Time
}

type ManifestsRepo struct {
	// URL to the manifests repository
	URL string

	// Git reference
	Ref Ref

	// Secret used to clone the repository
	CloneSecret string

	// Application strategy (e.g. kustomization, helm, raw)
	Strategy string

	// Path within the repository
	Path string
}

type Ref struct {
	Branch string
	Tag    string
	Commit string
}
