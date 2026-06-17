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

All Argo Events infrastructure — EventBus, EventSource, and Sensor — lives in
a single fixed namespace (argoEventsNamespace, "argo-events"), following Argo
Events' own convention of one shared stack per cluster rather than per-tenant
proliferation. GitHubEvent CRs are required to live in that same namespace:
Kubernetes does not support cross-namespace owner references for namespaced
resources, so EventSource/Sensor can only be owned by (and garbage-collected
with) a GitHubEvent CR that lives alongside them.

For each GitHubEvent CR the provider ensures three Argo resources exist,
all in argoEventsNamespace:
  - EventBus (not owned) — shared NATS bus. Argo Events resolves EventBus by
    looking for a bus named "default" within the same namespace as the
    EventSource/Sensor that depend on it — it does NOT resolve cross-namespace.
    Created once and left in place; never owned by any GitHubEvent CR.
  - EventSource (owned) — registers the GitHub webhook subscription and
    exposes the delivery endpoint.
  - Sensor (owned) — matches incoming payloads and emits GitHubEvent CRs via
    a Kubernetes trigger. The trigger's resource template targets
    githubEventAPIVersion — that must match whatever apiVersion the
    GitHubEvent CRD is actually served at, or the Sensor's create call 404s
    with "the server could not find the requested resource".

GitHubEvent's spec.contract field is intentionally opaque to Kubernetes
(x-kubernetes-preserve-unknown-fields) — the API server stores it verbatim
without validating its internal shape. Its actual structure is defined by
the GitHubEventSpec proto message, which uses snake_case field names
(event_type, commit_sha, webhook_secret_ref, etc.). The Sensor's trigger
must populate those exact proto field names.

Critically, Argo Events' Kubernetes trigger does NOT support inline Go-template
placeholders (e.g. "{{ .Input.body.X }}") embedded in arbitrary string fields
of the resource body — that syntax is not a real Argo Events feature for this
trigger type and gets stored verbatim as a literal string, never substituted.
The actual mechanism is the trigger's separate `parameters` array: a list of
src/dest mappings that extract values from the matched event's body/headers
(via dependencyName + dataKey) and inject them into specific JSON paths
(dest) within the resource AFTER the base resource is built. createGitHubSensor
builds an empty `contract` object as the base shape and lets `parameters` fill
it in — this is the only correct way to get live payload data into the
created GitHubEvent CR.

EventSource and Sensor carry an ownerReference to the GitHubEvent CR so they
are garbage-collected when the CR is deleted. If a GitHubEvent CR is ever
created outside argoEventsNamespace, Ensure() rejects it rather than silently
dropping the owner reference.

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

// argoEventsNamespace is the single namespace all Argo Events infrastructure
// lives in for this platform — EventBus, EventSource, and Sensor are always
// provisioned here, regardless of where any other resources in the cluster
// live. GitHubEvent CRs are expected to be created in this namespace too.
const argoEventsNamespace = "argo-events"

// githubEventAPIVersion is the apiVersion the Sensor's trigger uses when
// creating GitHubEvent CRs. Must match whatever version the CRD is actually
// served at — a mismatch here produces a 404 from the API server at trigger
// time, not at apply/install time, so it's easy to miss until a real event
// fires.
const githubEventAPIVersion = "events.blanketops.dev/v1alpha1"

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

// createEventBus constructs the NATS EventBus in argoEventsNamespace. The bus
// is shared across every GitHubEvent CR in the cluster — it is not owned by
// any single CR and is never deleted on CR removal.
func createEventBus() *unstructured.Unstructured {
	obj := newUnstructured("argoproj.io/v1alpha1", "EventBus", argoEventsNamespace, "default")
	unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"nats": map[string]interface{}{
			"native": map[string]interface{}{
				"replicas": int64(1), // unstructured requires int64/float64/string/bool/map/slice — plain int panics
			},
		},
	}, "spec")
	return obj
}

// createGitHubEventSource constructs the EventSource in argoEventsNamespace
// that registers the GitHub webhook subscription. Owned by the GitHubEvent
// CR — deleted when the CR is deleted.
func createGitHubEventSource(
	owner, repo string,
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
	obj.SetNamespace(argoEventsNamespace)
	unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"github": map[string]interface{}{
			"repo-events": map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"events":     eventList,
				"webhook": map[string]interface{}{
					"endpoint": "/github",
					// Argo Events' WebhookContext.Port is typed as a string in
					// their Go API, despite looking numeric. Sending a JSON
					// number here breaks decoding for this object AND poisons
					// the eventsource-controller's shared List/Watch — every
					// EventSource in the cluster fails to decode until this
					// object is fixed, since they share one typed watch stream.
					"port": "12000",
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

// createGitHubSensor constructs the Sensor in argoEventsNamespace that
// matches incoming GitHub payloads and emits GitHubEvent CRs via a
// Kubernetes trigger. Owned by the GitHubEvent CR — deleted when the CR is
// deleted.
//
// The trigger's resource defines only the base shape of spec.contract — an
// empty object. Real values are injected by the parameters block below,
// which maps fields out of the matched event's body/headers into specific
// dest paths in the created resource, using the GitHubEventSpec proto's
// exact snake_case field names. See the file-level doc comment for why
// inline templating in the resource body itself does not work here.
func createGitHubSensor() *unstructured.Unstructured {
	obj := newUnstructured("argoproj.io/v1alpha1", "Sensor", argoEventsNamespace, "github-sensor")
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
								"apiVersion": githubEventAPIVersion,
								"kind":       "GitHubEvent",
								"metadata": map[string]interface{}{
									"generateName": "github-event-",
									"namespace":    argoEventsNamespace,
								},
								"spec": map[string]interface{}{
									"contract": map[string]interface{}{},
								},
							},
						},
						"parameters": []interface{}{
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									"dataKey":        "body.repository.full_name",
								},
								"dest": "spec.contract.repository",
							},
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									// Header key casing: Go's HTTP header
									// canonicalization typically produces
									// "X-Github-Event" (lowercase 'h'), not
									// GitHub's own "X-GitHub-Event" casing.
									// Adjust here first if this parameter
									// comes back empty.
									"dataKey": "headers.X-Github-Event.0",
								},
								"dest": "spec.contract.event_type",
							},
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									"dataKey":        "body.ref",
								},
								"dest": "spec.contract.ref",
							},
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									"dataKey":        "body.after",
								},
								"dest": "spec.contract.commit_sha",
							},
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									"dataKey":        "body.sender.login",
								},
								"dest": "spec.contract.actor",
							},
							map[string]interface{}{
								"src": map[string]interface{}{
									"dependencyName": "github-dep",
									// Same casing caveat as event_type above.
									"dataKey": "headers.X-Github-Delivery.0",
								},
								"dest": "spec.contract.event_id",
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
// order, all in argoEventsNamespace. EventSource and Sensor are owned by the
// CR; EventBus is not.
//
// Returns Accepted on success and Rejected on the first apply failure, or if
// the GitHubEvent CR itself is not in argoEventsNamespace (since the owner
// reference cannot be established cross-namespace).
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

	bus := createEventBus()
	src := createGitHubEventSource(
		spec.Repository.Owner,
		spec.Repository.Name,
		[]string{string(spec.Type)},
		"github-webhook-secret",
		"secret",
	)
	sensor := createGitHubSensor()

	// EventSource and Sensor are owned — garbage-collected with the CR. This
	// only succeeds if cr itself lives in argoEventsNamespace; Kubernetes
	// rejects cross-namespace owner references for namespaced resources, so
	// a mismatch here surfaces as a hard error rather than a silently
	// dropped owner reference.
	if err := controllerutil.SetControllerReference(cr, src, p.Scheme); err != nil {
		return domain.Rejected(err.Error()), err
	}
	if err := controllerutil.SetControllerReference(cr, sensor, p.Scheme); err != nil {
		return domain.Rejected(err.Error()), err
	}

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
