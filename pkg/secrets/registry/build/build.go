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
Package build reconciles the Build registry credential secret — the
ExternalSecret backing the Build's image-pull ServiceAccount, and the
teardown of both that ExternalSecret and its ESO-materialized Secret when
the Build is deleted.
*/
package build

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

type BuildRegistryExternalSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

func NewBuildRegistryExternalSecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *BuildRegistryExternalSecretReconciler {
	return &BuildRegistryExternalSecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

func (r *BuildRegistryExternalSecretReconciler) Reconcile(ctx context.Context, build *buildResolution.ResolvedBuild) error {
	if build == nil || build.Build == nil || build.Spec == nil {
		return nil
	}
	if build.Spec.ServiceAccount == nil {
		return nil
	}

	secretName := build.Spec.ServiceAccount.Secret
	if secretName == "" {
		return nil
	}

	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": build.Build.Namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "registry",
					"blanketops.dev/build":   build.Build.Name,
				},
			},
			"spec": map[string]any{
				"refreshInterval": "10s",
				"secretStoreRef": map[string]any{
					"name": r.StoreName,
					"kind": r.StoreKind,
				},
				"target": map[string]any{
					"name": secretName,
					"template": map[string]any{
						"type": "kubernetes.io/dockerconfigjson",
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": ".dockerconfigjson",
						"remoteRef": map[string]any{
							"key": "/blanketops/registry/config",
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(build.Build, desired, r.Client.Scheme()); err != nil {
		return err
	}

	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(desired), &existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	r.Log.Info("creating ExternalSecret for registry credentials",
		"build", build.Build.Name, "secret", secretName, "store", r.StoreName)
	return r.Client.Create(ctx, desired)
}

// Delete removes both the ExternalSecret and the Secret ESO materialized
// from it. See BuildGitSSHSecretReconciler.Delete for why the Secret must be
// deleted explicitly rather than left to the ExternalSecret's
// ownerReference GC cascade — the same delete-then-recreate race applies
// here to the registry credentials secret.
func (r *BuildRegistryExternalSecretReconciler) Delete(ctx context.Context, build *buildResolution.ResolvedBuild) error {
	if build == nil || build.Build == nil || build.Spec == nil || build.Spec.ServiceAccount == nil {
		return nil
	}
	secretName := build.Spec.ServiceAccount.Secret
	if secretName == "" {
		return nil
	}
	namespace := build.Build.Namespace

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("external-secrets.io/v1")
	obj.SetKind("ExternalSecret")
	obj.SetName(secretName)
	obj.SetNamespace(namespace)
	if err := r.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{}
	secret.SetName(secretName)
	secret.SetNamespace(namespace)
	if err := r.Client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
