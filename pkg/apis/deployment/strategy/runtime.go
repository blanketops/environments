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
Package strategy dispatches a DeploymentIntent across the Runtime/Strategy
axis: which backend to deploy to (Kubernetes, ECS, Knative) and, once a
backend is chosen, which rollout strategy to run there (Rolling, BlueGreen).

In this domain, Kubernetes/ECS/Knative are themselves deployment strategies,
not a separate runtime layer sitting above strategy — so RuntimeProvider
(the Runtime switch) and each backend's own strategy dispatch (K8SStrategy's
Rolling/BlueGreen switch, plus the ECS/Knative placeholders) live together
in this one package rather than being split across "runtime" and "strategy"
packages.

This package only decides *which* strategy runs; the actual object
apply/teardown work is delegated to pkg/apis/deployment/api (e.g.
K8SStrategy wraps api.K8SProvider and calls its ApplyServiceUnit).
*/
package strategy

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/blanketops/environments/pkg/apis/deployment/api"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

// RuntimeProvider dispatches a DeploymentIntent's Runtime (Kubernetes,
// Knative, ECS) to the strategy that implements it. Runtime and Strategy
// are the same axis in this domain — which runtime to deploy to is itself
// a deployment strategy choice — so this and K8SStrategy live together in
// this package, not split across a separate "runtime" concept.
type RuntimeProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	K8S    *K8SStrategy
}

// NewRuntimeProvider constructs a RuntimeProvider, wiring up its K8S
// strategy from the given clients.
func NewRuntimeProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, Recorder events.EventRecorder) *RuntimeProvider {
	return &RuntimeProvider{
		Client: c,
		Scheme: scheme,
		Log:    log,
		K8S:    NewK8SStrategy(api.NewK8SProvider(c, scheme, log, Recorder), log),
	}
}

// Execute dispatches dIntent to the strategy implementing its Runtime.
// Knative and ECS are recognized but not yet implemented.
func (p *RuntimeProvider) Execute(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	switch dIntent.Runtime {

	case intent.RuntimeKubernetes:
		return p.K8S.Execute(ctx, dIntent)

	case intent.RuntimeKnative:
		return nil, fmt.Errorf("knative runtime not implemented")

	case intent.RuntimeECS:
		return nil, fmt.Errorf("ecs runtime not implemented")

	default:
		return nil, fmt.Errorf("unsupported runtime: %s", dIntent.Runtime)
	}
}

// Teardown dispatches dIntent to the strategy implementing its Runtime,
// mirroring Execute's dispatch. Knative and ECS are recognized but not yet
// implemented.
func (p *RuntimeProvider) Teardown(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) error {

	switch dIntent.Runtime {

	case intent.RuntimeKubernetes:
		return p.K8S.Teardown(ctx, dIntent)

	case intent.RuntimeKnative:
		return fmt.Errorf("knative runtime not implemented")

	case intent.RuntimeECS:
		return fmt.Errorf("ecs runtime not implemented")

	default:
		return fmt.Errorf("unsupported runtime: %s", dIntent.Runtime)
	}
}
