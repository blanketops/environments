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

package registry

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	packageResolution "github.com/ntlaletsi70/blanketops-environments/resolution/packages"
)

type PackageRegistrySecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
}

func NewPackageRegistrySecretReconciler(c client.Client, log logr.Logger, storeName string) *PackageRegistrySecretReconciler {
	return &PackageRegistrySecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
	}
}

func (r *PackageRegistrySecretReconciler) Reconcile(ctx context.Context, resolvedPackage *packageResolution.ResolvedPackage) error {
	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}

	pkg := resolvedPackage.Package
	secretName := resolvedPackage.Spec.PackageRepository.CredentialsSecret
	if secretName == "" {
		return nil
	}

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
					"name": r.StoreName,
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

	if err := controllerutil.SetControllerReference(pkg, desired, r.Client.Scheme()); err != nil {
		return err
	}

	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(desired), &existing)

	if apierrors.IsNotFound(err) {
		r.Log.Info("creating ExternalSecret for package registry credentials",
			"package", pkg.Name, "secret", secretName, "store", r.StoreName)
		return r.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	desiredSpec, _, _ := unstructured.NestedMap(desired.Object, "spec")
	existingSpec, _, _ := unstructured.NestedMap(existing.Object, "spec")

	if !reflect.DeepEqual(existingSpec, desiredSpec) {
		if err := unstructured.SetNestedMap(existing.Object, desiredSpec, "spec"); err != nil {
			return err
		}
		r.Log.Info("updating ExternalSecret for package registry credentials",
			"package", pkg.Name, "secret", secretName, "store", r.StoreName)
		return r.Client.Update(ctx, &existing)
	}

	r.Log.V(1).Info("ExternalSecret for package registry credentials already up-to-date",
		"package", pkg.Name, "secret", secretName)
	return nil
}

// registry/packageregistry.go — append
func (r *PackageRegistrySecretReconciler) Delete(ctx context.Context, resolvedPackage *packageResolution.ResolvedPackage) error {
	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}
	secretName := resolvedPackage.Spec.PackageRepository.CredentialsSecret
	if secretName == "" {
		return nil
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("external-secrets.io/v1")
	obj.SetKind("ExternalSecret")
	obj.SetName(secretName)
	obj.SetNamespace(resolvedPackage.Package.Namespace)
	if err := r.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
