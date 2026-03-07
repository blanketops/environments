package api

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type GitHubProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

func NewGitHubProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *GitHubProvider {
	return &GitHubProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

//
// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func newUnstructured(
	apiVersion, kind, namespace, name string,
) *unstructured.Unstructured {

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace(namespace)

	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		apiVersion,
		kind,
	))

	return obj
}

//
// -----------------------------------------------------------------------------
// Argo Events Resources
// -----------------------------------------------------------------------------

// EventBus is cluster-scoped and NOT owned
func createEventBus() *unstructured.Unstructured {
	obj := newUnstructured(
		"argoproj.io/v1alpha1",
		"EventBus",
		"default", // cluster-scoped
		"default",
	)

	unstructured.SetNestedMap(
		obj.Object,
		map[string]interface{}{
			"nats": map[string]interface{}{
				"native": map[string]interface{}{
					"replicas": int64(1), // 🔒 MUST NOT be int
				},
			},
		},
		"spec",
	)

	return obj
}

// EventSource is namespaced and OWNED
func createGitHubEventSource(
	namespace string,
	owner string,
	repo string,
	events []string,
	secretName string,
	secretKey string,
) *unstructured.Unstructured {

	// 🔒 Convert []string → []interface{}
	eventList := make([]interface{}, len(events))
	for i, e := range events {
		eventList[i] = e
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("argoproj.io/v1alpha1")
	obj.SetKind("EventSource")

	obj.SetName("github")
	obj.SetNamespace(namespace)

	unstructured.SetNestedMap(
		obj.Object,
		map[string]interface{}{
			"github": map[string]interface{}{
				"repo-events": map[string]interface{}{
					"owner":      owner,
					"repository": repo,
					"events":     eventList, // ✅ FIXED
					"webhook": map[string]interface{}{
						"endpoint": "/github",
						"port":     int64(12000), // 🔒 numeric fix
					},
					"webhookSecret": map[string]interface{}{
						"name": secretName,
						"key":  secretKey,
					},
					"active": true,
				},
			},
		},
		"spec",
	)

	return obj
}

// Sensor is namespaced and OWNED
func createGitHubSensor(namespace string) *unstructured.Unstructured {
	obj := newUnstructured(
		"argoproj.io/v1alpha1",
		"Sensor",
		namespace,
		"github-sensor",
	)

	obj.Object["spec"] = map[string]interface{}{
		"dependencies": []interface{}{
			map[string]interface{}{
				"name":            "github-dep",
				"eventSourceName": "github",
				"eventName":       "repo-events",
			},
		},
		"triggers": []interface{}{
			map[string]interface{}{
				"template": map[string]interface{}{
					"name": "emit-github-event",
					"k8s": map[string]interface{}{
						"operation": "create",
						"source": map[string]interface{}{
							"resource": map[string]interface{}{
								"apiVersion": "events.blanketops.dev/v1",
								"kind":       "GitHubEvent",
								"metadata": map[string]interface{}{
									"generateName": "github-event-",
									"namespace":    namespace,
								},
								"spec": map[string]interface{}{
									"repository": "{{ .Input.body.repository.full_name }}",
									"eventType":  "{{ .Input.headers.X-GitHub-Event }}",
									"ref":        "{{ .Input.body.ref }}",
									"commitSHA":  "{{ .Input.body.after }}",
									"actor":      "{{ .Input.body.sender.login }}",
								},
							},
						},
					},
				},
			},
		},
	}

	return obj
}

//
// -----------------------------------------------------------------------------
// Provider Logic
// -----------------------------------------------------------------------------

func (p *GitHubProvider) Ensure(
	ctx context.Context,
	resolved *githubeventResolution.ResolvedGitHubEvent,
	spec domain.GitHubEvent,
) (domain.GitHubEventResult, error) {

	cr := resolved.Event

	p.Log.Info(
		"github.ensure: ensuring ingress",
		"repo", spec.Repository.FullName,
		"type", spec.Type,
	)

	// Shared infra
	bus := createEventBus()

	// Owned infra
	src := createGitHubEventSource(
		cr.Namespace,
		spec.Repository.Owner,
		spec.Repository.Name,
		[]string{string(spec.Type)},
		"github-webhook-secret",
		"secret",
	)

	sensor := createGitHubSensor(cr.Namespace)

	// Ownership: delete with GitHubEvent
	_ = controllerutil.SetControllerReference(cr, src, p.Scheme)
	_ = controllerutil.SetControllerReference(cr, sensor, p.Scheme)

	for _, obj := range []*unstructured.Unstructured{
		bus,
		src,
		sensor,
	} {
		if err := p.apply(ctx, obj); err != nil {
			return domain.Rejected(err.Error()), err
		}
	}

	return domain.Accepted("github ingress ensured"), nil
}

func (p *GitHubProvider) apply(
	ctx context.Context,
	obj *unstructured.Unstructured,
) error {

	// ------------------------------------------------------------------
	// HARD GVK ASSERTION — NEVER REMOVE
	// ------------------------------------------------------------------
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		obj.GetAPIVersion(),
		obj.GetKind(),
	))

	// ------------------------------------------------------------------
	// OBJECT KEY — CLUSTER VS NAMESPACED
	// ------------------------------------------------------------------
	var key client.ObjectKey

	if obj.GetNamespace() == "" {
		// Cluster-scoped resource (e.g. EventBus)
		key = client.ObjectKey{
			Name: obj.GetName(),
		}
	} else {
		// Namespaced resource (e.g. EventSource, Sensor)
		key = client.ObjectKey{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}
	}

	// ------------------------------------------------------------------
	// FETCH CURRENT
	// ------------------------------------------------------------------
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(obj.GroupVersionKind())

	err := p.Client.Get(ctx, key, current)
	switch {

	// ---- HARD ERROR ----
	case client.IgnoreNotFound(err) != nil:
		p.Log.Error(err, "apply: get failed", "key", key, "gvk", obj.GroupVersionKind())
		return err

	// ---- CREATE ----
	case err != nil:
		p.Log.Info("apply: creating", "gvk", obj.GroupVersionKind(), "key", key)

		if err := p.Client.Create(ctx, obj); err != nil {
			p.Log.Error(err, "apply: create failed", "key", key)
			return err
		}

		if p.Recorder != nil {
			p.Recorder.Eventf(
				obj,
				nil,
				"Normal",
				"Created",
				"%s %s created",
				obj.GetKind(),
				key.Name,
			)
		}

		return nil

	// ---- UPDATE ----
	default:
		obj.SetResourceVersion(current.GetResourceVersion())

		p.Log.Info("apply: updating", "gvk", obj.GroupVersionKind(), "key", key)

		if err := p.Client.Update(ctx, obj); err != nil {
			p.Log.Error(err, "apply: update failed", "key", key)
			return err
		}

		if p.Recorder != nil {
			p.Recorder.Eventf(
				obj,
				nil,
				"Normal",
				"Updated",
				"%s %s updated",
				obj.GetKind(),
				key.Name,
			)
		}

		return nil
	}
}
