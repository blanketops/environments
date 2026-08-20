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
Package build reconciles the Build git-ssh clone secret — the ExternalSecret
used to clone the Build's source/state repository over SSH, and the
teardown of both that ExternalSecret and its ESO-materialized Secret when
the Build is deleted.
*/
package build

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

// BuildGitSSHSecretReconciler converges the ExternalSecret used to clone a
// Build's source repository over SSH, and tears it down (plus its
// ESO-materialized Secret) on Build deletion.
type BuildGitSSHSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

// NewBuildGitSSHSecretReconciler constructs a BuildGitSSHSecretReconciler
// targeting the given ESO store.
func NewBuildGitSSHSecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *BuildGitSSHSecretReconciler {
	return &BuildGitSSHSecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

// Reconcile creates or updates the git-ssh clone ExternalSecret for build's
// source repository.
func (r *BuildGitSSHSecretReconciler) Reconcile(ctx context.Context, build *buildResolution.ResolvedBuild) error {

	source := build.Spec.Source
	secretName := source.CloneSecret
	namespace := build.Build.Namespace

	// -------------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// Keys are platform constants — never change regardless of provider.
	// Only secretStoreRef.name is provider-dependent.
	// -------------------------------------------------------------------------
	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "git-ssh",
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
						"type": "kubernetes.io/ssh-auth",
					},
				},
				"data": []any{
					map[string]any{
						"secretKey": "ssh-privatekey",
						"remoteRef": map[string]any{
							"key": "/blanketops/git/ssh-privatekey",
						},
					},
					map[string]any{
						"secretKey": "ssh-publickey",
						"remoteRef": map[string]any{
							"key": "/blanketops/git/ssh-publickey",
						},
					},
					map[string]any{
						"secretKey": "known_hosts",
						"remoteRef": map[string]any{
							"key": "/blanketops/git/known-hosts",
						},
					},
				},
			},
		},
	}

	// -------------------------------------------------------------------------
	// Ownership (Build → ExternalSecret)
	// -------------------------------------------------------------------------
	if err := controllerutil.SetControllerReference(
		build.Build,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Fetch existing
	// -------------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      secretName,
			Namespace: namespace,
		},
		&existing,
	)

	// -------------------------------------------------------------------------
	// Create
	// -------------------------------------------------------------------------
	if apierrors.IsNotFound(err) {
		r.Log.Info(
			"creating ExternalSecret for Git SSH",
			"build", build.Build.Name,
			"secret", secretName,
			"store", r.StoreName,
		)
		return r.Client.Create(ctx, desired)
	}

	if err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Update (spec drift only)
	// -------------------------------------------------------------------------
	if !reflect.DeepEqual(
		existing.Object["spec"],
		desired.Object["spec"],
	) {
		existing.Object["spec"] = desired.Object["spec"]

		r.Log.Info(
			"updating ExternalSecret for Git SSH",
			"build", build.Build.Name,
			"secret", secretName,
			"store", r.StoreName,
		)

		return r.Client.Update(ctx, &existing)
	}

	// -------------------------------------------------------------------------
	// No-op
	// -------------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret already up-to-date",
		"build", build.Build.Name,
		"secret", secretName,
	)

	return nil
}

// Delete removes both the ExternalSecret and the Secret ESO materialized
// from it. The Secret carries an ownerReference back to the ExternalSecret,
// so it would eventually be removed by Kubernetes' garbage collector on its
// own — but that GC cascade runs on its own async reconcile loop and is not
// bounded by this call returning. A caller that deletes a Build and
// immediately recreates it (same name, same derived secretName) can lose
// the race: the new ExternalSecret's target Secret collides with the old
// Secret still awaiting GC, and ESO refuses to adopt a Secret it doesn't
// own, leaving the new Build without a working secret until GC catches up
// and ESO retries. Deleting the Secret directly here closes that window —
// plain Secrets carry no finalizers, so this delete is synchronous from the
// API server's perspective.
func (r *BuildGitSSHSecretReconciler) Delete(ctx context.Context, build *buildResolution.ResolvedBuild) error {
	secretName := build.Spec.Source.CloneSecret
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
