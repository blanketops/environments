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

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package domain

import "time"

/*
DOMAIN PRINCIPLES

- No Kubernetes imports
- No controller-runtime
- No side effects
- No persistence concerns

This file represents the *intent* and *result* of a Deployment,
not how it is reconciled.
*/

// DeploymentSpec represents the desired state of a Deployment.
// This is the canonical domain input used by application logic.
type DeploymentSpec struct {
	Name string

	ServiceUnits []string

	Runtime Runtime

	// Rollout strategy (Rolling / Canary / BlueGreen)
	Strategy Strategy

	ImageAutomation bool

	ManifestsRepo *ManifestsRepo

	GitOwner string

	ReconciliationStrategy ReconciliationStrategy
}

type ManifestsRepo struct {
	URL         string
	Ref         string
	CloneSecret string
	Strategy    string
	Path        string
}

type ReconciliationStrategy string

const (
	ReconciliationImperative ReconciliationStrategy = "Imperative"
	ReconciliationKustomize  ReconciliationStrategy = "Kustomize"
	ReconciliationHelm       ReconciliationStrategy = "Helm"
)

type Runtime string

const (
	RuntimeKubernetes Runtime = "kubernetes.io/container-runtime"
	RuntimeKnative    Runtime = "knative.dev/service-runtime"
	RuntimeWasm       Runtime = "blanketops.dev/wasm-runtime"
	RuntimeECS        Runtime = "blanketops.dev/aws-ecs"
	RuntimeAzure      Runtime = "blanketops.dev/azure-container"
)

type Strategy string

const (
	StrategyRolling   Strategy = "Rolling"
	StrategyBlueGreen Strategy = "BlueGreen"
	StrategyCanary    Strategy = "Canary"
)

type DeploymentPhase string
type ServiceUnitPhase string

//
// ---- Deployment Result ----
//

// DeploymentResult represents the outcome of executing a DeploymentIntent
type DeploymentResult struct {
	Phase DeploymentPhase

	Message string

	Runtime  Runtime
	Strategy Strategy

	ServiceUnits []ServiceUnitResult

	LastUpdateTime time.Time
}

type ServiceUnitResult struct {
	Name string

	Phase ServiceUnitPhase

	Image string

	Runtime Runtime

	Message string
	Error   string

	LastTransitionTime time.Time
}

//
// ---- Errors ----
//
