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

package git

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

type PackageStateRepositorySecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewPackageStateRepositorySecretReconciler(
	c client.Client,
	log logr.Logger,
) *PackageStateRepositorySecretReconciler {
	return &PackageStateRepositorySecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *PackageStateRepositorySecretReconciler) Reconcile(
	ctx context.Context,
	resolvedPackage *packageResolution.ResolvedPackage,
) error {

	if resolvedPackage == nil || resolvedPackage.Package == nil {
		return nil
	}

	pkg := resolvedPackage.Package

	// 🔑 Secret name is authoritative from Package contract
	secretName := resolvedPackage.Spec.StateRepository.CloneSecret
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
						"remoteRef": map[string]any{
							"key": "/blanketops/git/ssh-privatekey",
						},
					},
					map[string]any{
						"secretKey": "ssh_publickey",
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
			"Creating ExternalSecret for package state repository (git ssh)",
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
			"Updating ExternalSecret for package state repository (git ssh)",
			"package", pkg.Name,
			"secret", secretName,
		)

		return r.Client.Update(ctx, &existing)
	}

	// ---------------------------------------------------------------------
	// NO-OP
	// ---------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret for package state repository already up-to-date",
		"package", pkg.Name,
		"secret", secretName,
	)

	return nil
}
