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
Copyright 2025.
*/

package domain

//
// ---- Deployment State ----
//

// DeploymentState represents the high-level lifecycle state
type DeploymentState string

const (
	DeploymentPending     DeploymentState = "Pending"
	DeploymentReconciling DeploymentState = "Reconciling"
	DeploymentDeploying   DeploymentState = "Deploying"
	DeploymentReady       DeploymentState = "Ready"
	DeploymentDegraded    DeploymentState = "Degraded"
	DeploymentFailed      DeploymentState = "Failed"
)

//
// ---- ServiceUnit State ----
//

type ServiceUnitState string

const (
	ServiceUnitPending   ServiceUnitState = "Pending"
	ServiceUnitResolving ServiceUnitState = "Resolving"
	ServiceUnitDeploying ServiceUnitState = "Deploying"
	ServiceUnitReady     ServiceUnitState = "Ready"
	ServiceUnitFailed    ServiceUnitState = "Failed"
)

//
// ---- State Derivation ----
//

// StateFromResult derives a DeploymentState from a Result
func StateFromResult(r Result) DeploymentState {
	if r.Success {
		return DeploymentReady
	}

	if r.Partial {
		return DeploymentDegraded
	}

	if r.Error != nil {
		switch r.Error.Kind {
		case ErrInvalidSpec, ErrMissingReference:
			return DeploymentFailed
		case ErrBuildNotReady:
			return DeploymentReconciling
		default:
			return DeploymentFailed
		}
	}

	return DeploymentFailed
}

// ServiceUnitStateFromResult derives ServiceUnitState from ServiceUnitResult
// ServiceUnitStateFromResult derives ServiceUnitState from ServiceUnitResult
func ServiceUnitStateFromResult(r ServiceUnitResult) ServiceUnitState {
	return ServiceUnitState(r.Phase)
}
