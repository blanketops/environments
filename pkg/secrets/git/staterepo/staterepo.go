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
Package staterepo reconciles the Package state-repository clone secret —
the same ExternalSecret lifecycle as the Build/Deployment variants
(pkg/secrets/git/build, pkg/secrets/git/deployment), scoped to a Package's
declared state repository. Named staterepo rather than package: "package"
is a reserved Go keyword.
*/
package staterepo

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

// PackageStateRepositorySecretReconciler converges the ExternalSecret used
// to clone a Package's declared state repository — the same lifecycle as
// BuildGitSSHSecretReconciler, scoped to Package.
type PackageStateRepositorySecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

// NewPackageStateRepositorySecretReconciler constructs a
// PackageStateRepositorySecretReconciler targeting the given ESO store.
func NewPackageStateRepositorySecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *PackageStateRepositorySecretReconciler {
	return &PackageStateRepositorySecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

// Reconcile creates or updates the git-ssh clone ExternalSecret for
// resolvedPackage's state repository.
func (r *PackageStateRepositorySecretReconciler) Reconcile(ctx context.Context, resolvedPackage *packageResolution.ResolvedPackage) error {
	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}

	pkg := resolvedPackage.Package
	secretName := resolvedPackage.Spec.StateRepository.CloneSecret
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
				"refreshInterval": "10s",
				"secretStoreRef": map[string]any{
					"name": r.StoreName,
					"kind": r.StoreKind,
				},
				"target": map[string]any{
					"name": secretName,
					"template": map[string]any{
						"type": "Opaque",
						"data": map[string]any{
							"ssh-privatekey": "{{ .ssh_privatekey }}",
							"ssh-publickey":  "{{ .ssh_publickey }}",
							"known_hosts":    "{{ .known_hosts }}",
						},
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": "ssh_privatekey",
						"remoteRef": map[string]any{"key": "/blanketops/git/ssh-privatekey"},
					},
					map[string]any{
						"secretKey": "ssh_publickey",
						"remoteRef": map[string]any{"key": "/blanketops/git/ssh-publickey"},
					},
					map[string]any{
						"secretKey": "known_hosts",
						"remoteRef": map[string]any{"key": "/blanketops/git/known-hosts"},
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
		r.Log.Info("creating ExternalSecret for package state repository",
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
		r.Log.Info("updating ExternalSecret for package state repository",
			"package", pkg.Name, "secret", secretName, "store", r.StoreName)
		return r.Client.Update(ctx, &existing)
	}

	r.Log.V(1).Info("ExternalSecret for package state repository already up-to-date",
		"package", pkg.Name, "secret", secretName)
	return nil
}

// Delete removes both the ExternalSecret and the Secret ESO materialized
// from it. See BuildGitSSHSecretReconciler.Delete for why the Secret must be
// deleted explicitly rather than left to the ExternalSecret's ownerReference
// GC cascade.
func (r *PackageStateRepositorySecretReconciler) Delete(ctx context.Context, resolvedPackage *packageResolution.ResolvedPackage) error {
	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}
	secretName := resolvedPackage.Spec.StateRepository.CloneSecret
	if secretName == "" {
		return nil
	}
	namespace := resolvedPackage.Package.Namespace

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
