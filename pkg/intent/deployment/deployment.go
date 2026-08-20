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
Package deployment owns DeploymentIntent — the Kubernetes-free, fully
resolved plan for a Deployment, built once its ResolvedDeployment and
ResolvedServiceUnits are known. It carries which runtime and strategy to
use, the reconciliation mode, and each ServiceUnit's own
pkg/intent/serviceunit.ServiceUnitIntent.

IntentBuilder.Build (intent_builder.go) is the canonical, complete
constructor and the one pkg/apis/deployment/application.DeploymentService
actually calls. ResolveDeploymentIntent (resolve.go) is an older, partial
constructor that never learned Runtime/Strategy/ReconciliationStrategy/
ManifestsRepo — it has no callers in DeploymentService's path and should be
treated as superseded rather than a second supported entry point.
*/
package deployment

import (
	"time"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

// DeploymentIntent is the Kubernetes-free, fully resolved plan for a
// Deployment: which runtime and strategy to use, the reconciliation mode,
// and each ServiceUnit's own intent. Built by IntentBuilder.Build.
type DeploymentIntent struct {
	Source *environmentv1alpha1.Deployment // ← IMPORTANT

	Name      string
	Namespace string

	Runtime  Runtime
	Strategy Strategy

	ServiceUnits []serviceunitIntent.ServiceUnitIntent

	ReconciliationStrategy ReconciliationStrategy
	ImageAutomation        bool
	ManifestsRepo          *ManifestsRepo

	GitOwner    string
	GeneratedAt time.Time
}

// ManifestsRepo is the Git repository a Deployment's manifests are rendered
// from or applied against.
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

// Ref pins ManifestsRepo to a specific branch, tag, or commit.
type Ref struct {
	Branch string
	Tag    string
	Commit string
}
