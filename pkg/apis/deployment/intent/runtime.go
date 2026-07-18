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

type Runtime string

const (
	RuntimeKubernetes Runtime = "kubernetes.io/container-runtime"
	RuntimeKnative    Runtime = "knative.dev/service-runtime"
	RuntimeWASM       Runtime = "blanketops.dev/wasm-runtime"
	RuntimeECS        Runtime = "blanketops.dev/aws-ecs"
	RuntimeAzure      Runtime = "blanketops.dev/azure-container"
)

type Strategy string

const (
	StrategyRolling   Strategy = "Rolling"
	StrategyBlueGreen Strategy = "BlueGreen"
	StrategyCanary    Strategy = "Canary"
)
