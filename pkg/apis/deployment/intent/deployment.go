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
	"time"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
)

type DeploymentIntent struct {
	Source *environmentv1alpha1.Deployment // ← IMPORTANT

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
