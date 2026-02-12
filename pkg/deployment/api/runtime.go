package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments-mvp/pkg/deployment/intent"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RuntimeProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	K8S    *K8SProvider
}

func NewRuntimeProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, Recorder record.EventRecorder) *RuntimeProvider {
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
