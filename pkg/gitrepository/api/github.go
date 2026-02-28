package api

import (
	"context"

	"github.com/go-logr/logr"
	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"

	"github.com/ntlaletsi70/blanketops-environments/pkg/gitrepository/domain"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//
// -----------------------------------------------------------------------------
// Provider
// -----------------------------------------------------------------------------

type GitHubProvider struct {
	Client   ctrlclient.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder // optional
}

func NewGitHubProvider(
	c ctrlclient.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec record.EventRecorder,
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
// Crossplane Repository (cluster-scoped)
// -----------------------------------------------------------------------------

func CreateRepositorySpec(
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) *unstructured.Unstructured {

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "repo.github.upbound.io/v1alpha1",
			"kind":       "Repository",
			"metadata": map[string]interface{}{
				"name": cr.Name,
				"labels": map[string]interface{}{
					"sources.blanketops.dev/gitrepository": cr.Name,
					"sources.blanketops.dev/provider":      spec.Provider,
				},
			},
			"spec": map[string]interface{}{
				"forProvider": map[string]interface{}{
					"name":       spec.Repository.Name,
					"visibility": "private",
				},
				"providerConfigRef": map[string]interface{}{
					"name": "github-upjet",
				},
			},
		},
	}

	// HARD GVK ASSERTION
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		"repo.github.upbound.io/v1alpha1",
		"Repository",
	))

	return obj
}

//
// -----------------------------------------------------------------------------
// Crossplane RepositoryWebhook (cluster-scoped)
// -----------------------------------------------------------------------------

func CreateRepositoryWebhookSpec(
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) *unstructured.Unstructured {

	events := make([]interface{}, 0)
	for _, wh := range spec.Webhooks {
		for _, e := range wh.Events {
			// Crossplane expects string event names
			events = append(events, e)
		}
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "repo.github.upbound.io/v1alpha1",
			"kind":       "RepositoryWebhook",
			"metadata": map[string]interface{}{
				"name": cr.Name + "-webhook",
				"labels": map[string]interface{}{
					"sources.blanketops.dev/gitrepository": cr.Name,
					"sources.blanketops.dev/provider":      spec.Provider,
				},
			},
			"spec": map[string]interface{}{
				"forProvider": map[string]interface{}{
					"active": true,
					"events": events,
					"configuration": []interface{}{
						map[string]interface{}{
							"urlSecretRef": map[string]interface{}{
								"name":      "hookurl",
								"key":       "url",
								"namespace": cr.Namespace,
							},
						},
					},
					"repositoryRef": map[string]interface{}{
						"name": cr.Name,
					},
				},
				"providerConfigRef": map[string]interface{}{
					"name": "github-upjet",
				},
			},
		},
	}

	// HARD GVK ASSERTION
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		"repo.github.upbound.io/v1alpha1",
		"RepositoryWebhook",
	))

	return obj
}

//
// -----------------------------------------------------------------------------
// Provider orchestration
// -----------------------------------------------------------------------------

func (p *GitHubProvider) Ensure(
	ctx context.Context,
	cr *sourcesv1alpha1.GitRepository,
	spec domain.GitRepository,
) (domain.Result, error) {

	// ---- 0. Validate domain intent ----
	if err := spec.Validate(); err != nil {
		return domain.Failure(err.Error()), err
	}

	p.Log.Info(
		"provider.ensure: reconciling GitHub repository",
		"gitrepository", ctrlclient.ObjectKeyFromObject(cr),
	)

	// -------------------------
	// 1. Repository
	// -------------------------
	repo := CreateRepositorySpec(cr, spec)
	if err := p.apply(ctx, repo); err != nil {
		return domain.Failure(err.Error()), err
	}

	// -------------------------
	// 2. RepositoryWebhook (optional)
	// -------------------------
	if len(spec.Webhooks) > 0 {
		hook := CreateRepositoryWebhookSpec(cr, spec)
		if err := p.apply(ctx, hook); err != nil {
			return domain.Failure(err.Error()), err
		}
	}

	p.Log.Info(
		"provider.ensure: reconciliation complete",
		"repository", cr.Name,
	)

	return domain.Success(), nil
}

//
// -----------------------------------------------------------------------------
// Apply helper (cluster-scoped, REST-mapper safe)
// -----------------------------------------------------------------------------

// NOTE:
// Crossplane-managed resources MUST be applied via Server-Side Apply.
// Never use Update() here — it will race the Crossplane controllers.

func (p *GitHubProvider) apply(
	ctx context.Context,
	obj *unstructured.Unstructured,
) error {

	// HARD GVK ASSERTION — NEVER REMOVE
	obj.SetGroupVersionKind(schema.FromAPIVersionAndKind(
		obj.GetAPIVersion(),
		obj.GetKind(),
	))

	p.Log.Info(
		"provider.apply: applying (ssa)",
		"gvk", obj.GroupVersionKind(),
		"name", obj.GetName(),
	)

	// IMPORTANT:
	// - Do NOT set ResourceVersion
	// - Do NOT Get() first
	// - Let Kubernetes merge fields safely
	obj.SetManagedFields(nil)

	return p.Client.Patch(
		ctx,
		obj,
		ctrlclient.Apply,
		ctrlclient.ForceOwnership,
		ctrlclient.FieldOwner("blanketops"),
	)
}
