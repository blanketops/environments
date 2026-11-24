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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/gitrepository/domain"
	"github.com/blanketops/environments/pkg/secrets/github/hookurl"
	"github.com/blanketops/environments/pkg/secrets/github/webhooksecret"
)

const fieldOwner = "blanketops"

// Compile-time assertion — GitHubProvider must satisfy Provider.
var _ Provider = (*GitHubProvider)(nil)

// ── Provider ──────────────────────────────────────────────────────────────────

// GitHubProvider implements Provider against the upjet-github Crossplane
// provider, ensuring the Repository and RepositoryWebhook resources that
// back a GitRepository CR.
type GitHubProvider struct {
	Client   ctrlclient.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

// NewGitHubProvider constructs a GitHubProvider.
func NewGitHubProvider(
	c ctrlclient.Client,
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

// ── Builders ──────────────────────────────────────────────────────────────────

func repositoryResourceName(cr *sourcesv1alpha1.GitRepository) string {
	return cr.Name
}

func webhookResourceName(cr *sourcesv1alpha1.GitRepository) string {
	return cr.Name + "-webhook"
}

func targetRepoName(cr *sourcesv1alpha1.GitRepository, spec domain.GitRepository) string {
	if spec.Repository.Name != "" {
		return spec.Repository.Name
	}
	return cr.Name
}

func buildRepository(
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) *Repository {
	return &Repository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "Repository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: repositoryResourceName(cr),
			Labels: map[string]string{
				"sources.blanketops.dev/gitrepository": cr.Name,
				"sources.blanketops.dev/provider":      string(spec.Provider),
			},
		},
		Spec: RepositorySpec{
			ProviderConfigRef: ProviderConfigRef{
				Name: "github-upjet",
			},
			ForProvider: RepositoryParameters{
				Name:       ptr(targetRepoName(cr, spec)),
				Visibility: ptr("private"),
			},
		},
	}
}

func buildRepositoryWebhook(
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) *RepositoryWebhook {
	seen := make(map[string]struct{})
	events := make([]*string, 0)
	for _, wh := range spec.Webhooks {
		for _, e := range wh.Events {
			ev := string(e)
			if ev == "" {
				continue
			}
			if _, exists := seen[ev]; exists {
				continue
			}
			seen[ev] = struct{}{}
			events = append(events, ptr(ev))
		}
	}

	return &RepositoryWebhook{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "RepositoryWebhook",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookResourceName(cr),
			Labels: map[string]string{
				"sources.blanketops.dev/gitrepository": cr.Name,
				"sources.blanketops.dev/provider":      string(spec.Provider),
			},
		},
		Spec: RepositoryWebhookSpec{
			ProviderConfigRef: ProviderConfigRef{
				Name: "github-upjet",
			},
			ForProvider: RepositoryWebhookParameters{
				Active:     ptr(true),
				Events:     events,
				Repository: ptr(targetRepoName(cr, spec)),
				Configuration: []RepositoryWebhookConfigurationParameters{
					{
						URLSecretRef: SecretKeyRef{
							Name:      hookurl.HookURLSecretName(cr.Name),
							Namespace: cr.Namespace,
							Key:       hookurl.HookURLSecretKey,
						},
						// json so Argo Events can address body.* fields directly;
						// GitHub's default "form" encoding wraps the payload.
						ContentType: ptr("json"),
						// HMAC secret so GitHub signs deliveries. Sourced from the
						// same store key the eventsource validates against, so the
						// signatures match. See pkg/secrets/github/webhooksecret.
						SecretSecretRef: &SecretKeyRef{
							Name:      webhooksecret.SecretName(cr.Name),
							Namespace: cr.Namespace,
							Key:       webhooksecret.SecretKey,
						},
					},
				},
			},
		},
	}
}

// ── Ensure ────────────────────────────────────────────────────────────────────

// Ensure reconciles the upjet-github Repository (and, if webhooks are
// declared, RepositoryWebhook) resources backing cr against spec.
func (p *GitHubProvider) Ensure(
	ctx context.Context,
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) (domain.Result, error) {
	if cr == nil || cr.Name == "" {
		return domain.Failure("GitRepository CR or name is empty"), fmt.Errorf("ensure gitrepository: GitRepository CR or name is empty")
	}

	if err := spec.Validate(); err != nil {
		p.recordEvent(cr, corev1.EventTypeWarning, "InvalidSpec", err.Error())
		return domain.Failure(err.Error()), err
	}

	p.Log.Info(
		"github.ensure: reconciling repository",
		"gitrepository", ctrlclient.ObjectKeyFromObject(cr),
		"repository", targetRepoName(cr, spec),
	)

	// 1. Repository
	repo := buildRepository(cr, spec)
	if err := p.apply(ctx, repo); err != nil {
		p.recordEvent(cr, corev1.EventTypeWarning, "EnsureRepositoryFailed", err.Error())
		return domain.Failure(err.Error()), err
	}

	// 2. RepositoryWebhook — apply when declared, or remove if previously created and now empty.
	if len(spec.Webhooks) > 0 {
		webhook := buildRepositoryWebhook(cr, spec)
		if err := p.apply(ctx, webhook); err != nil {
			p.recordEvent(cr, corev1.EventTypeWarning, "EnsureWebhookFailed", err.Error())
			return domain.Failure(err.Error()), err
		}
	} else {
		if err := p.deleteIfExists(ctx, &RepositoryWebhook{
			TypeMeta: metav1.TypeMeta{
				APIVersion: SchemeGroupVersion.String(),
				Kind:       "RepositoryWebhook",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookResourceName(cr),
			},
		}); err != nil {
			p.recordEvent(cr, corev1.EventTypeWarning, "CleanupWebhookFailed", err.Error())
			return domain.Failure(err.Error()), err
		}
	}

	p.Log.Info(
		"github.ensure: reconciliation complete",
		"gitrepository", ctrlclient.ObjectKeyFromObject(cr),
		"repository", targetRepoName(cr, spec),
	)
	p.recordEvent(cr, corev1.EventTypeNormal, "Ensured", "GitHub repository resources reconciled")

	return domain.Success(), nil
}

// ── Apply ─────────────────────────────────────────────────────────────────────

// NOTE: Crossplane-managed resources MUST be applied via SSA.
// Never use Update() — it will race the Crossplane controllers.
func (p *GitHubProvider) apply(ctx context.Context, obj ctrlclient.Object) error {
	if obj == nil {
		return fmt.Errorf("apply object: nil object")
	}

	p.Log.Info(
		"github.apply: applying (ssa)",
		"gvk", obj.GetObjectKind().GroupVersionKind(),
		"name", obj.GetName(),
	)

	// Clone before mutating metadata so the caller's struct isn't unexpectedly altered.
	patchObj, ok := obj.DeepCopyObject().(ctrlclient.Object)
	if !ok || patchObj == nil {
		return fmt.Errorf("apply object %s: failed to deep copy object", obj.GetName())
	}
	patchObj.SetManagedFields(nil)

	return p.Client.Patch(
		ctx,
		patchObj,
		ctrlclient.Apply, //nolint:staticcheck
		ctrlclient.ForceOwnership,
		ctrlclient.FieldOwner(fieldOwner),
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

func (p *GitHubProvider) recordEvent(
	cr *sourcesv1alpha1.GitRepository,
	eventType string,
	reason string,
	note string,
) {
	if p.Recorder == nil || cr == nil {
		return
	}
	p.Recorder.Eventf(cr, nil, eventType, reason, reason, note)
}

func (p *GitHubProvider) deleteIfExists(ctx context.Context, obj ctrlclient.Object) error {
	if err := p.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// Teardown deletes the Crossplane Repository and RepositoryWebhook this
// provider created. Mandatory, not optional — Ensure links these objects to
// the GitRepository CR by label only (sources.blanketops.dev/gitrepository),
// never by ownerReference, and both are cluster-scoped. Kubernetes GC has no
// mechanism to reclaim them; without this, deleting the GitRepository CR
// leaves Crossplane managing a live GitHub repository and webhook forever.
//
// Idempotent — a missing Repository or RepositoryWebhook is not an error.
func (p *GitHubProvider) Teardown(
	ctx context.Context,
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) error {
	p.Log.Info(
		"github.teardown: removing repository",
		"gitrepository", ctrlclient.ObjectKeyFromObject(cr),
	)

	// 1. RepositoryWebhook first — release the hook before the repo it
	//    targets, so Crossplane isn't left reconciling a webhook against a
	//    Repository that's mid-deletion.
	if len(spec.Webhooks) > 0 {
		webhook := &RepositoryWebhook{
			TypeMeta: metav1.TypeMeta{
				APIVersion: SchemeGroupVersion.String(),
				Kind:       "RepositoryWebhook",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: cr.Name + "-webhook",
			},
		}
		if err := p.Client.Delete(ctx, webhook); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete repository webhook: %w", err)
		}
	}

	// 2. Repository.
	repo := &Repository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "Repository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: repositoryResourceName(cr),
		},
	}
	if err := p.deleteIfExists(ctx, repo); err != nil {
		p.recordEvent(cr, corev1.EventTypeWarning, "TeardownRepositoryFailed", err.Error())
		return fmt.Errorf("delete repository: %w", err)
	}

	p.Log.Info("github.teardown: complete", "repository", repositoryResourceName(cr))
	p.recordEvent(cr, corev1.EventTypeNormal, "TeardownComplete", "GitHub repository resources deleted")
	return nil
}
