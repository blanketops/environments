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
This file owns GitHubProvider — the concrete provider that provisions the Argo
Events stack required to receive GitHub webhook deliveries and emit GitHubEvent
CRs into the cluster.

For each GitHubEvent CR the provider ensures three Argo resources exist:

  - EventBus (cluster-scoped, not owned) — shared NATS bus. Created once and
    left in place; never owned by a GitHubEvent CR.
  - EventSource (namespaced, owned) — registers the GitHub webhook subscription
    and exposes the delivery endpoint.
  - Sensor (namespaced, owned) — matches incoming payloads and emits
    GitHubEvent CRs via a Kubernetes trigger.

EventSource and Sensor carry an ownerReference to the GitHubEvent CR so they
are garbage-collected when the CR is deleted. EventBus is cluster-scoped and
intentionally not owned.

apply() is the shared upsert primitive — it creates or updates any Unstructured
object with correct GVK assertion and ResourceVersion threading.
*/
package api

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

// GitHubProvider provisions and maintains the Argo Events stack for a
// GitHubEvent CR. It is the only component in the platform that writes
// Argo Events API objects directly.
type GitHubProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder // optional; may be nil
}

// NewGitHubProvider constructs a GitHubProvider with the given dependencies.
func NewGitHubProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *GitHubProvider {
	return &GitHubProvider{Client: c, Scheme: scheme, Log: log, Recorder: rec}
}

// -----------------------------------------------------------------------------
// Argo resource constructors
// -----------------------------------------------------------------------------

// newUnstructured is a helper that constructs a bare Unstructured object with
// apiVersion, kind, namespace, and name set. GVK is derived from apiVersion
// and kind so Kubernetes API machinery can route the request correctly.
func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(apiVersion, kind))
	return obj
}

// createEventBus constructs the namespaced NATS EventBus. The bus is shared
// across all GitHubEvent CRs within a namespace — it is not owned by any
// single CR and is never deleted on CR removal, but it must live in the same
// namespace as the EventSource/Sensor that depend on it (Argo Events does not
// resolve EventBus cross-namespace).
func createEventBus(namespace string) *unstructured.Unstructured {
	obj := newUnstructured("argoproj.io/v1alpha1", "EventBus", namespace, "default")
	unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"nats": map[string]interface{}{
			"native": map[string]interface{}{
				"replicas": int64(1), // must be int64 — JSON numbers decode as float64
			},
		},
	}, "spec")
	return obj
}

// createGitHubEventSource constructs the namespaced EventSource that registers
// the GitHub webhook subscription. Owned by the GitHubEvent CR — deleted when
// the CR is deleted.
func createGitHubEventSource(
	namespace, owner, repo string,
	events []string,
	secretName, secretKey string,
) *unstructured.Unstructured {
	// Unstructured requires []interface{} — convert from []string.
	eventList := make([]interface{}, len(events))
	for i, e := range events {
		eventList[i] = e
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("argoproj.io/v1alpha1")
	obj.SetKind("EventSource")
	obj.SetName("github")
	obj.SetNamespace(namespace)

	unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"github": map[string]interface{}{
			"repo-events": map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"events":     eventList,
				"webhook": map[string]interface{}{
					"endpoint": "/github",
					"port":     int64(12000), // must be int64
				},
				"webhookSecret": map[string]interface{}{
					"name": secretName,
					"key":  secretKey,
				},
				"active": true,
			},
		},
	}, "spec")
	return obj
}

// createGitHubSensor constructs the namespaced Sensor that matches incoming
// GitHub payloads and emits GitHubEvent CRs via a Kubernetes trigger.
// Owned by the GitHubEvent CR — deleted when the CR is deleted.
func createGitHubSensor(namespace string) *unstructured.Unstructured {
	obj := newUnstructured("argoproj.io/v1alpha1", "Sensor", namespace, "github-sensor")
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

// -----------------------------------------------------------------------------
// Provider logic
// -----------------------------------------------------------------------------

// Ensure provisions or reconciles the full Argo Events stack for the given
// GitHubEvent CR. It creates or updates EventBus, EventSource, and Sensor in
// order. EventSource and Sensor are owned by the CR; EventBus is not.
//
// Returns Accepted on success and Rejected on the first apply failure.
func (p *GitHubProvider) Ensure(
	ctx context.Context,
	resolved *githubeventResolution.ResolvedGitHubEvent,
	spec domain.GitHubEvent,
) (domain.GitHubEventResult, error) {
	cr := resolved.Event

	p.Log.Info("github.ensure: ensuring ingress",
		"repo", spec.Repository.FullName,
		"type", spec.Type,
	)

	bus := createEventBus(cr.Namespace)
	src := createGitHubEventSource(
		cr.Namespace,
		spec.Repository.Owner,
		spec.Repository.Name,
		[]string{string(spec.Type)},
		"github-webhook-secret",
		"secret",
	)
	sensor := createGitHubSensor(cr.Namespace)

	// EventSource and Sensor are owned — garbage-collected with the CR.
	// EventBus is cluster-scoped and intentionally not owned.
	_ = controllerutil.SetControllerReference(cr, src, p.Scheme)
	_ = controllerutil.SetControllerReference(cr, sensor, p.Scheme)

	for _, obj := range []*unstructured.Unstructured{bus, src, sensor} {
		if err := p.apply(ctx, obj); err != nil {
			return domain.Rejected(err.Error()), err
		}
	}

	return domain.Accepted("github ingress ensured"), nil
}

// apply creates or updates any Unstructured object. It asserts GVK before
// every API call — the Kubernetes client requires GVK to be set on
// Unstructured objects and it must not be lost across assignments.
//
// Cluster-scoped resources (empty namespace) use a name-only ObjectKey.
// ResourceVersion is threaded from the existing object on update to satisfy
// the optimistic concurrency requirement.
func (p *GitHubProvider) apply(ctx context.Context, obj *unstructured.Unstructured) error {
	// GVK must be set before every API call — never remove this.
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(obj.GetAPIVersion(), obj.GetKind()))

	var key client.ObjectKey
	if obj.GetNamespace() == "" {
		key = client.ObjectKey{Name: obj.GetName()}
	} else {
		key = client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	}

	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(obj.GroupVersionKind())
	err := p.Client.Get(ctx, key, current)

	switch {
	case client.IgnoreNotFound(err) != nil:
		p.Log.Error(err, "apply: get failed", "key", key, "gvk", obj.GroupVersionKind())
		return err

	case err != nil: // not found — create
		p.Log.Info("apply: creating", "gvk", obj.GroupVersionKind(), "key", key)
		if err := p.Client.Create(ctx, obj); err != nil {
			p.Log.Error(err, "apply: create failed", "key", key)
			return err
		}
		if p.Recorder != nil {
			p.Recorder.Eventf(obj, nil, "Normal", "Created", "%s %s created", obj.GetKind(), key.Name)
		}
		return nil

	default: // found — update
		obj.SetResourceVersion(current.GetResourceVersion())
		p.Log.Info("apply: updating", "gvk", obj.GroupVersionKind(), "key", key)
		if err := p.Client.Update(ctx, obj); err != nil {
			p.Log.Error(err, "apply: update failed", "key", key)
			return err
		}
		if p.Recorder != nil {
			p.Recorder.Eventf(obj, nil, "Normal", "Updated", "%s %s updated", obj.GetKind(), key.Name)
		}
		return nil
	}
}
