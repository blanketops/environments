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
Package deployment defines ReconciliationStrategy enum (Imperative, Kustomize, Helm).

See also:
  - ./pkg/intent/deployment/reconciliation.go/../resolution/.../adapter.go — produces resolved input for this intent
*/
package deployment

// ReconciliationStrategy selects how a Deployment's manifests are applied
// (imperative apply, Kustomize, or Helm).
type ReconciliationStrategy string

const (
	ReconciliationImperative ReconciliationStrategy = "Imperative"
	ReconciliationKustomize  ReconciliationStrategy = "Kustomize"
	ReconciliationHelm       ReconciliationStrategy = "Helm"
)

