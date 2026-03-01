package secrets

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GitHubProviderConfigReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewGitHubProviderConfigReconciler(
	c client.Client,
	log logr.Logger,
) *GitHubProviderConfigReconciler {
	return &GitHubProviderConfigReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *GitHubProviderConfigReconciler) Reconcile(ctx context.Context) error {
	const (
		name      = "github-upjet"
		namespace = "crossplane-system"

		// This secret is produced by the ExternalSecret reconciler
		secretName = "example-creds"
		secretKey  = "credentials"
	)

	desired := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "github.upbound.io/v1beta1",
			"kind":       "ProviderConfig",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "crossplane-github-provider",
				},
			},
			"spec": map[string]interface{}{
				"credentials": map[string]interface{}{
					"source": "Secret",
					"secretRef": map[string]interface{}{
						"name":      secretName,
						"namespace": namespace,
						"key":       secretKey,
					},
				},
			},
		},
	}

	desired.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "github.upbound.io",
		Version: "v1beta1",
		Kind:    "ProviderConfig",
	})

	key := client.ObjectKey{Name: name}
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, key, current)
	switch {
	case apierrors.IsNotFound(err):
		r.Log.Info("Creating GitHub ProviderConfig", "name", name)
		return r.Client.Create(ctx, desired)

	case err != nil:
		return fmt.Errorf("get ProviderConfig: %w", err)

	default:
		// Update spec only, keep resourceVersion for safe update
		desired.SetResourceVersion(current.GetResourceVersion())

		r.Log.Info("Updating GitHub ProviderConfig", "name", name)
		return r.Client.Update(ctx, desired)
	}
}
