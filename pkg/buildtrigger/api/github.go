package api

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BuildTriggerProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

func NewBuildTriggerProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *BuildTriggerProvider {
	return &BuildTriggerProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}
