package api

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BuildTriggerProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

func NewBuildTriggerProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec record.EventRecorder,
) *BuildTriggerProvider {
	return &BuildTriggerProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

func createBuildRun(
	namespace string,
	trigger domain.BuildTrigger,
) *unstructured.Unstructured {

	name := fmt.Sprintf(
		"buildrun-%s",
		trigger.Trigger.ID[:8], // deterministic, short
	)

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("shipwright.io/v1beta1")
	obj.SetKind("BuildRun")
	obj.SetNamespace(namespace)
	obj.SetName(name)

	obj.Object["spec"] = map[string]interface{}{
		"buildRef": map[string]interface{}{
			"name": trigger.Target.Name,
		},
		"parameters": []interface{}{
			map[string]interface{}{
				"name":  "GIT_REF",
				"value": trigger.Trigger.Ref,
			},
			map[string]interface{}{
				"name":  "GIT_SHA",
				"value": trigger.Trigger.SHA,
			},
		},
	}

	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		"shipwright.io/v1beta1",
		"BuildRun",
	))

	return obj
}
