package registry

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	packageResolution "github.com/ntlaletsi70/blanketops-environments/resolution/packages"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type PackageRegistrySecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewPackageRegistrySecretReconciler(
	c client.Client,
	log logr.Logger,
) *PackageRegistrySecretReconciler {
	return &PackageRegistrySecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *PackageRegistrySecretReconciler) Reconcile(
	ctx context.Context,
	resolvedPackage *packageResolution.ResolvedPackage,
) error {

	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}

	pkg := resolvedPackage.Package

	// 🔑 Secret name is authoritative from Package contract
	secretName := resolvedPackage.Spec.PackageRepository.CredentialsSecret
	if secretName == "" {
		return nil
	}

	// ---------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// ---------------------------------------------------------------------
	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": pkg.Namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/domain":  "package",
					"blanketops.dev/package": pkg.Name,
				},
			},
			"spec": map[string]any{
				"refreshInterval": "0s",
				"secretStoreRef": map[string]any{
					"name": "blanketops-environments-fake",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]any{
					"name": secretName,
					"template": map[string]any{
						"type": "kubernetes.io/dockerconfigjson",
						"data": map[string]any{
							".dockerconfigjson": "{{ .dockerconfigjson }}",
						},
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": "dockerconfigjson",
						"remoteRef": map[string]any{
							"key": "/blanketops/registry/config",
						},
					},
				},
			},
		},
	}

	// 🔑 OWN the ExternalSecret (never the Secret)
	if err := controllerutil.SetControllerReference(
		pkg,
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
			"Creating ExternalSecret for package registry credentials",
			"package", pkg.Name,
			"secret", secretName,
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
			"Updating ExternalSecret for package registry credentials",
			"package", pkg.Name,
			"secret", secretName,
		)

		return r.Client.Update(ctx, &existing)
	}

	// ---------------------------------------------------------------------
	// NO-OP
	// ---------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret for package registry credentials already up-to-date",
		"package", pkg.Name,
		"secret", secretName,
	)

	return nil
}
