package secrets

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	HookURLSecretName = "hookurl"
	HookURLSecretKey  = "url"
)

type HookURLExternalSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewHookURLExternalSecretReconciler(
	c client.Client,
	log logr.Logger,
) *HookURLExternalSecretReconciler {
	return &HookURLExternalSecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *HookURLExternalSecretReconciler) Reconcile(
	ctx context.Context,
	repo *sourcesv1alpha1.GitRepository,
) error {

	if repo == nil {
		return fmt.Errorf("nil GitRepository provided to HookURLExternalSecretReconciler")
	}

	remoteKey := fmt.Sprintf(
		"/blanketops/sources/%s/hookurl",
		repo.Name,
	)

	// ---------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// ---------------------------------------------------------------------
	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1beta1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      HookURLSecretName,
				"namespace": repo.Namespace,
				"labels": map[string]any{
					"sources.blanketops.dev/gitrepository": repo.Name,
					"blanketops.dev/managed":               "true",
					"blanketops.dev/purpose":               "webhook",
				},
			},
			"spec": map[string]any{
				"refreshInterval": "0s",
				"secretStoreRef": map[string]any{
					"name": "blanketops-environments-fake",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]any{
					"name": HookURLSecretName,
				},
				"data": []any{
					map[string]any{
						"secretKey": HookURLSecretKey,
						"remoteRef": map[string]any{
							"key": remoteKey,
						},
					},
				},
			},
		},
	}

	// 🔑 OWN the ExternalSecret (never the Secret)
	if err := controllerutil.SetControllerReference(
		repo,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	// ---------------------------------------------------------------------
	// Fetch existing
	// ---------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKeyFromObject(desired),
		&existing,
	)

	// ---------------------------------------------------------------------
	// CREATE
	// ---------------------------------------------------------------------
	if apierrors.IsNotFound(err) {
		r.Log.Info(
			"Creating ExternalSecret for GitRepository hook URL",
			"repository", repo.Name,
			"namespace", repo.Namespace,
		)
		return r.Client.Create(ctx, desired)
	}

	if err != nil {
		return err
	}

	// ---------------------------------------------------------------------
	// UPDATE (spec drift)
	// ---------------------------------------------------------------------
	desiredSpec, _, _ := unstructured.NestedMap(desired.Object, "spec")
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")

	if !reflect.DeepEqual(existingSpec, desiredSpec) {
		if err := unstructured.SetNestedMap(
			existing.Object,
			desiredSpec,
			"spec",
		); err != nil {
			return err
		}

		r.Log.Info(
			"Updating ExternalSecret for GitRepository hook URL",
			"repository", repo.Name,
			"namespace", repo.Namespace,
		)

		return r.Client.Update(ctx, &existing)
	}

	// ---------------------------------------------------------------------
	// NO-OP
	// ---------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret for GitRepository hook URL already up-to-date",
		"repository", repo.Name,
	)

	return nil
}
