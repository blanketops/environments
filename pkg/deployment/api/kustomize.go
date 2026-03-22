package api

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KustomizationProvider struct {
	Client                       client.Client
	Scheme                       *runtime.Scheme
	Log                          logr.Logger
	NewKustomizeStrategyProvider *KustomizeStrategyProvider
}

func NewKustomizationProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, Recorder record.EventRecorder) *KustomizeStrategyProvider {
	return &KustomizeStrategyProvider{
		Client: c,
		Scheme: scheme,
		Log:    log,
	}
}
