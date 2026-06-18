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
	"encoding/json"

	argoeventsv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ntlaletsi70/blanketops-environments/pkg/githubevent/domain"
	githubeventResolution "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
)

const argoEventsNamespace = "argo-events"

// GitHubProvider is the concrete Provider implementation for GitHubEvent.
// It provisions the Argo Events stack (EventBus, EventSource, Sensor)
// required to receive and forward GitHub webhook deliveries.
type GitHubProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

func NewGitHubProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, rec events.EventRecorder) *GitHubProvider {
	return &GitHubProvider{Client: c, Scheme: scheme, Log: log, Recorder: rec}
}

// Ensure provisions or reconciles the Argo Events stack for the GitHubEvent CR.
func (p *GitHubProvider) Ensure(
	ctx context.Context,
	resolved *githubeventResolution.ResolvedGitHubEvent,
	spec domain.GitHubEvent,
) (domain.GitHubEventResult, error) {
	cr := resolved.Event

	bus := p.createTypedEventBus()
	src := p.createTypedGitHubEventSource(spec)
	sensor, err := p.createTypedGitHubSensor(spec, cr.Name)
	if err != nil {
		p.Log.Error(err, "ensure: failed building sensor spec", "name", cr.Name)
		return domain.Rejected(err.Error()), err
	}

	// Set OwnerRefs for garbage collection. EventBus is intentionally
	// excluded — it's a shared cluster-wide singleton, not owned by any
	// single GitHubEvent CR.
	for _, obj := range []client.Object{src, sensor} {
		if err := controllerutil.SetControllerReference(cr, obj, p.Scheme); err != nil {
			p.Log.Error(err, "ensure: failed setting owner reference", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
			return domain.Rejected(err.Error()), err
		}
	}

	for _, obj := range []client.Object{bus, src, sensor} {
		if err := p.apply(ctx, obj); err != nil {
			p.Log.Error(err, "ensure: failed applying object", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
			return domain.Rejected(err.Error()), err
		}
	}
	return domain.Accepted("github ingress ensured"), nil
}

func (p *GitHubProvider) createTypedEventBus() *argoeventsv1alpha1.EventBus {
	replicas := int32(1)
	return &argoeventsv1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: argoEventsNamespace},
		Spec: argoeventsv1alpha1.EventBusSpec{
			NATS: &argoeventsv1alpha1.NATSBus{
				Native: &argoeventsv1alpha1.NativeStrategy{Replicas: replicas},
			},
		},
	}
}

func (p *GitHubProvider) createTypedGitHubEventSource(spec domain.GitHubEvent) *argoeventsv1alpha1.EventSource {
	owner, repoName := spec.Repository.Owner, spec.Repository.Name
	return &argoeventsv1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: argoEventsNamespace},
		Spec: argoeventsv1alpha1.EventSourceSpec{
			Github: map[string]argoeventsv1alpha1.GithubEventSource{
				"repo-events": {
					Repositories: []argoeventsv1alpha1.OwnedRepositories{
						{Owner: owner, Names: []string{repoName}},
					},
					Events:  []string{string(spec.Type)},
					Webhook: &argoeventsv1alpha1.WebhookContext{Endpoint: "/github", Port: "12000"},
					WebhookSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-webhook-secret"},
						Key:                  "secret",
					},
					Active: true,
				},
			},
		},
	}
}

// createTypedGitHubSensor builds the Sensor that watches for GitHub webhook
// deliveries and, on match, creates an immutable ConfigMap audit record of
// the payload. Every TriggerParameter must set DependencyName — Argo Events'
// sensor-controller validates this even for parameters that only inject a
// static Value rather than pulling from event data.
func (p *GitHubProvider) createTypedGitHubSensor(spec domain.GitHubEvent, crName string) (*argoeventsv1alpha1.Sensor, error) {
	basePayload := map[string]interface{}{
		"apiVersion": "events.blanketops.dev/v1alpha1",
		"kind":       "GitHubPayload",
		"metadata": map[string]interface{}{
			"generateName": "github-payload-",
			"namespace":    argoEventsNamespace,
		},
		"spec": map[string]interface{}{
			"contract": map[string]interface{}{},
		},
	}
	basePayloadBytes, err := json.Marshal(basePayload)
	if err != nil {
		return nil, err
	}
	return &argoeventsv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "github-sensor", Namespace: argoEventsNamespace},
		Spec: argoeventsv1alpha1.SensorSpec{
			Dependencies: []argoeventsv1alpha1.EventDependency{{
				Name:            "github-dep",
				EventSourceName: "github",
				EventName:       "repo-events",
				Filters: &argoeventsv1alpha1.EventDependencyFilter{
					Data: []argoeventsv1alpha1.DataFilter{{
						Path:  "headers.X-Github-Event",
						Type:  "string",
						Value: []string{string(spec.Type)},
					}},
				},
			}},
			Triggers: []argoeventsv1alpha1.Trigger{{
				Template: &argoeventsv1alpha1.TriggerTemplate{
					Name: "emit-github-payload",
					K8s: &argoeventsv1alpha1.StandardK8STrigger{
						Operation: "create",
						Source: &argoeventsv1alpha1.ArtifactLocation{
							Resource: &argoeventsv1alpha1.K8SResource{Value: basePayloadBytes},
						},
						// dest paths are spec.contract.X — opaque to
						// Kubernetes, decoded by ResolveGitHubPayload using
						// these exact camelCase keys.
						Parameters: []argoeventsv1alpha1.TriggerParameter{
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataTemplate: `{{ index (index .Input.headers "X-Github-Event") 0 }}`}, Dest: "spec.contract.eventType"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataTemplate: `{{ index (index .Input.headers "X-Github-Delivery") 0 }}`}, Dest: "spec.contract.eventId"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataKey: "body.repository.full_name"}, Dest: "spec.contract.repository"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataKey: "body.ref"}, Dest: "spec.contract.ref"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataKey: "body.after"}, Dest: "spec.contract.commitSHA"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", DataKey: "body.sender.login"}, Dest: "spec.contract.actor"},
							{Src: &argoeventsv1alpha1.TriggerParameterSource{DependencyName: "github-dep", Value: &crName}, Dest: "spec.contract.githubEventRef"},
						},
					},
				},
			}},
		},
	}, nil
}
func (p *GitHubProvider) apply(ctx context.Context, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	current := obj.DeepCopyObject().(client.Object)
	err := p.Client.Get(ctx, key, current)

	if client.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		p.Log.Info("apply: creating", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
		if err := p.Client.Create(ctx, obj); err != nil {
			return err
		}
		if p.Recorder != nil {
			p.Recorder.Eventf(obj, nil, "Normal", "Created", "Created %s", obj.GetName())
		}
		return nil
	}

	obj.SetResourceVersion(current.GetResourceVersion())
	p.Log.Info("apply: updating", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
	if err := p.Client.Update(ctx, obj); err != nil {
		return err
	}
	if p.Recorder != nil {
		p.Recorder.Eventf(obj, nil, "Normal", "Updated", "Updated %s", obj.GetName())
	}
	return nil
}
