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

package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	"github.com/blanketops/environments/pkg/apis/deployment/intent"
)

type RuntimeProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	K8S    *K8SProvider
}

func NewRuntimeProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, Recorder events.EventRecorder) *RuntimeProvider {
	return &RuntimeProvider{
		Client: c,
		Scheme: scheme,
		Log:    log,
		K8S:    NewK8SProvider(c, scheme, log, Recorder), // ⚡ initialize here
	}
}

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
