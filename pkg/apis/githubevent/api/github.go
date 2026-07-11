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
	"fmt"

	argoeventsv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	githubeventResolution "github.com/BlanketOps/environments/resolution/githubevent"
	"github.com/BlanketOps/environments/pkg/apis/githubevent/domain"
)

const argoEventsNamespace = "argo-events"

// githubSensorServiceAccount is the dedicated ServiceAccount the GitHub
// Sensor runs as. Scoped narrowly to GitHubEvent write access — the Sensor
// has no business touching anything else in the cluster.
const githubSensorServiceAccount = "github-sensor-sa"

// githubEventWriterRole / githubEventWriterRoleBinding name the RBAC objects
// granting the Sensor SA permission to create/patch GitHubEvent CRs.
const (
	githubEventWriterRole        = "githubevent-writer"
	githubEventWriterRoleBinding = "argo-events-githubevent-writer"
)

// GitHubProvider is the concrete Provider implementation for GitHubEvent.
// It provisions the Argo Events stack (EventBus, EventSource, Sensor) and
// the RBAC the Sensor needs to write GitHubEvent CRs.
type GitHubProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

func NewGitHubProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, rec events.EventRecorder) *GitHubProvider {
	return &GitHubProvider{Client: c, Scheme: scheme, Log: log, Recorder: rec}
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

// createTypedSensorServiceAccount is the identity the github-sensor Pod runs
// as. Created separately from the Role/RoleBinding so it can be referenced
// by name on the Sensor's pod template before RBAC is necessarily applied
// in the same reconcile pass — apply() is idempotent either way.
func (p *GitHubProvider) createTypedSensorServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      githubSensorServiceAccount,
			Namespace: argoEventsNamespace,
		},
	}
}

// createTypedGitHubEventRole grants exactly the verbs the Sensor's patch
// trigger needs against GitHubEvent — nothing else. Tighten further to
// resourceNames if the Sensor ever stops needing to create new CRs (it
// currently does, per createTypedGitHubSensor's patch-with-create-fallback
// semantics).
func (p *GitHubProvider) createTypedGitHubEventRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      githubEventWriterRole,
			Namespace: argoEventsNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"events.blanketops.dev"},
				Resources: []string{"githubevents"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
			},
		},
	}
}

func (p *GitHubProvider) createTypedGitHubEventRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      githubEventWriterRoleBinding,
			Namespace: argoEventsNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      githubSensorServiceAccount,
				Namespace: argoEventsNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     githubEventWriterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

// createTypedGitHubSensor builds the Sensor that watches for GitHub webhook
// deliveries and, on match, creates an immutable ConfigMap audit record of
// the payload. Every TriggerParameter must set DependencyName — Argo Events'
// sensor-controller validates this even for parameters that only inject a
// static Value rather than pulling from event data.
//
// The Sensor runs as githubSensorServiceAccount — set explicitly rather than
// left to default, so the RBAC grant in createTypedGitHubEventRoleBinding
// actually applies to the identity the patch trigger executes as.
func (p *GitHubProvider) createTypedGitHubSensor(spec domain.GitHubEvent, crName string) (*argoeventsv1alpha1.Sensor, error) {
	sensorName := "github-sensor-" + crName
	basePayload := map[string]interface{}{
		"apiVersion": "events.blanketops.dev/v1alpha1",
		"kind":       "GitHubEvent",
		"metadata": map[string]interface{}{
			"name":      crName,
			"namespace": argoEventsNamespace,
			"labels": map[string]interface{}{
				"events.blanketops.dev/githubevent": crName,
			},
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      sensorName,
			Namespace: argoEventsNamespace,
			Labels: map[string]string{
				"events.blanketops.dev/githubevent": crName,
			},
		},
		Spec: argoeventsv1alpha1.SensorSpec{
			Template: &argoeventsv1alpha1.Template{
				ServiceAccountName: githubSensorServiceAccount,
			},
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
					Name: "github-payload",
					K8s: &argoeventsv1alpha1.StandardK8STrigger{
						Operation: "patch",
						Source: &argoeventsv1alpha1.ArtifactLocation{
							Resource: &argoeventsv1alpha1.K8SResource{Value: basePayloadBytes},
						},
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

// Ensure provisions or reconciles the Argo Events stack — plus the RBAC the
// Sensor needs — for the GitHubEvent CR.
//
// Ensure does NOT wait for a webhook delivery. Infrastructure provisioning
// completing successfully is signaled via Triggered=true — the actual
// payload-received outcome is written later by the observer/reconcile path
// that detects the Sensor's patch on this CR's own spec.contract.
func (p *GitHubProvider) Ensure(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent, spec domain.GitHubEvent) (domain.GitHubEventResult, error) {
	cr := resolved.Event
	res := domain.GitHubEventResult{
		Success: false,
		Message: "",
	}

	p.Log.Info(
		"provider.evaluate: starting orchestration",
		"githubevent", client.ObjectKeyFromObject(resolved.Event).String(),
		"eventype", spec.Type,
	)

	// ------------------------------------------------
	// Stage 1: Build the typed objects.
	// ------------------------------------------------
	bus := p.createTypedEventBus()
	src := p.createTypedGitHubEventSource(spec)
	sa := p.createTypedSensorServiceAccount()
	role := p.createTypedGitHubEventRole()
	roleBinding := p.createTypedGitHubEventRoleBinding()

	sensor, err := p.createTypedGitHubSensor(spec, cr.Name)
	if err != nil {
		res.Message = err.Error()
		return res, err
	}

	// ------------------------------------------------
	// Stage 2: Owner references. EventBus is intentionally excluded — it's
	// a shared cluster-wide singleton, not owned by any single GitHubEvent
	// CR. The RBAC trio (sa/role/roleBinding) follows the same lifecycle as
	// the Sensor they support, so they're owned alongside it.
	// ------------------------------------------------
	for _, obj := range []client.Object{src, sensor, sa, role, roleBinding} {
		if err := controllerutil.SetControllerReference(cr, obj, p.Scheme); err != nil {
			res.Message = err.Error()
			return res, err
		}
	}

	// ------------------------------------------------
	// Stage 3: Apply (upsert) each object. RBAC and the EventSource land
	// before the Sensor, since the Sensor's pod template references the SA
	// by name and its trigger depends on the EventSource existing. First
	// failure aborts.
	// ------------------------------------------------
	for _, obj := range []client.Object{bus, sa, role, roleBinding, src, sensor} {
		if err := p.apply(ctx, obj); err != nil {
			p.Log.Error(err, "ensure: failed applying object", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
			res.Message = err.Error()
			return res, err
		}
	}

	// ------------------------------------------------
	// Infrastructure ensured — return intent result.
	//
	// Triggered=true mirrors Run(): the ingress infrastructure was
	// successfully created or already existed. Success stays false — this
	// only confirms infrastructure is ready, not that a payload arrived.
	// ------------------------------------------------
	res.Success = true
	res.Triggered = true        // Infrastructure is ready
	res.PayloadRecieved = false // NOTHING arrived yet
	res.Message = "GitHubEvent infrastructure provisioned"

	p.Log.Info(
		"provider.ensure: orchestration complete",
		"githubevent", cr.Name,
		"sensor", sensor.Name,
	)

	return res, nil
}

func (p *GitHubProvider) apply(ctx context.Context, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	current := obj.DeepCopyObject().(client.Object)
	err := p.Client.Get(ctx, key, current)
	if client.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		p.Log.Info("apply: creating", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
		return p.Client.Create(ctx, obj)
	}
	obj.SetResourceVersion(current.GetResourceVersion())
	p.Log.Info("apply: updating", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
	return p.Client.Update(ctx, obj)
}

// Teardown deletes the Sensor this GitHubEvent CR owns.
//
// EventBus, EventSource, the Sensor ServiceAccount, and its RBAC are shared
// cluster singletons — provisioned once and reused across every GitHubEvent
// CR — so they are NOT deleted here. Only the Sensor is genuinely per-CR
// (named "github-sensor-" + crName); deleting the shared objects would break
// every other GitHubEvent still depending on them.
//
// Idempotent — a missing Sensor is not an error.
func (p *GitHubProvider) Teardown(ctx context.Context, resolved *githubeventResolution.ResolvedGitHubEvent) error {
	cr := resolved.Event

	sensor := &argoeventsv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-sensor-" + cr.Name,
			Namespace: argoEventsNamespace,
		},
	}

	p.Log.Info("provider.teardown: deleting sensor", "githubevent", cr.Name, "sensor", sensor.Name)

	if err := p.Client.Delete(ctx, sensor); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete sensor %s: %w", sensor.Name, err)
	}

	return nil
}
