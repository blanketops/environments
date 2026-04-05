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

	"github.com/go-logr/logr"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type BuildRegistryExternalSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewBuildRegistryExternalSecretReconciler(
	c client.Client,
	log logr.Logger,
) *BuildRegistryExternalSecretReconciler {
	return &BuildRegistryExternalSecretReconciler{
		Client: c,
		Log:    log,
	}
}

// Reconcile ensures an ExternalSecret exists for registry credentials
// requested by the Build contract.
func (r *BuildRegistryExternalSecretReconciler) Reconcile(
	ctx context.Context,
	build *buildResolution.ResolvedBuild,
) error {

	if build == nil || build.Build == nil || build.Spec == nil {
		return nil
	}

	spec := build.Spec

	if spec.ServiceAccount == nil {
		// No service account → no registry auth requested
		return nil
	}

	secretName := spec.ServiceAccount.Secret
	if secretName == "" {
		// ServiceAccount exists but registry secret not requested
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
				"namespace": build.Build.Namespace,
				"labels": map[string]any{
					"blanketops.dev/managed": "true",
					"blanketops.dev/purpose": "registry",
					"blanketops.dev/build":   build.Build.Name,
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

	// 🔑 OWN the ExternalSecret (never the Secret)
	if err := controllerutil.SetControllerReference(
		build.Build,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	// ---------------------------------------------------------------------
	// Fetch existing (CREATE-ONLY semantics)
	// ---------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKeyFromObject(desired),
		&existing,
	)

	if err == nil {
		// Contract says: existence is enough
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	r.Log.Info(
		"Creating ExternalSecret for registry credentials",
		"build", build.Build.Name,
		"secret", secretName,
	)

	return r.Client.Create(ctx, desired)
}
