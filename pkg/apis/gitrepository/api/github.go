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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/gitrepository/domain"
)

// Compile-time assertion — GitHubProvider must satisfy Provider.
var _ Provider = (*GitHubProvider)(nil)

// ── Provider ──────────────────────────────────────────────────────────────────

type GitHubProvider struct {
	Client   ctrlclient.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder
}

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
			Name: cr.Name,
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
				Name:       ptr(spec.Repository.Name),
				Visibility: ptr("private"),
			},
		},
	}
}

func buildRepositoryWebhook(
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) *RepositoryWebhook {
	events := make([]*string, 0)
	for _, wh := range spec.Webhooks {
		for _, e := range wh.Events {
			events = append(events, ptr(string(e)))
		}
	}

	return &RepositoryWebhook{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "RepositoryWebhook",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Name + "-webhook",
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
				Repository: ptr(cr.Name),
				Configuration: []RepositoryWebhookConfigurationParameters{
					{
						URLSecretRef: SecretKeyRef{
							Name:      "hookurl",
							Namespace: cr.Namespace,
							Key:       "url",
						},
					},
				},
			},
		},
	}
}

// ── Ensure ────────────────────────────────────────────────────────────────────

func (p *GitHubProvider) Ensure(
	ctx context.Context,
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) (domain.Result, error) {

	if err := spec.Validate(); err != nil {
		return domain.Failure(err.Error()), err
	}

	p.Log.Info(
		"github.ensure: reconciling repository",
		"gitrepository", ctrlclient.ObjectKeyFromObject(cr),
	)

	// 1. Repository
	if err := p.apply(ctx, buildRepository(cr, spec)); err != nil {
		return domain.Failure(err.Error()), err
	}

	// 2. RepositoryWebhook — only when webhooks are declared
	if len(spec.Webhooks) > 0 {
		if err := p.apply(ctx, buildRepositoryWebhook(cr, spec)); err != nil {
			return domain.Failure(err.Error()), err
		}
	}

	p.Log.Info(
		"github.ensure: reconciliation complete",
		"repository", cr.Name,
	)

	return domain.Success(), nil
}

// ── Apply ─────────────────────────────────────────────────────────────────────

// NOTE: Crossplane-managed resources MUST be applied via SSA.
// Never use Update() — it will race the Crossplane controllers.
func (p *GitHubProvider) apply(ctx context.Context, obj ctrlclient.Object) error {
	p.Log.Info(
		"github.apply: applying (ssa)",
		"gvk", obj.GetObjectKind().GroupVersionKind(),
		"name", obj.GetName(),
	)

	obj.SetManagedFields(nil)

	return p.Client.Patch(
		ctx,
		obj,
		ctrlclient.Apply,
		ctrlclient.ForceOwnership,
		ctrlclient.FieldOwner("blanketops"),
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }
