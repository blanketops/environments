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

package github

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
)

const (
	HookURLSecretName = "hookurl"
	HookURLSecretKey  = "url"
)

type HookURLExternalSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewHookURLExternalSecretReconciler(c client.Client, log logr.Logger) *HookURLExternalSecretReconciler {
	return &HookURLExternalSecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *HookURLExternalSecretReconciler) Reconcile(ctx context.Context, repo *sourcesv1alpha1.GitRepository) error {

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
			"apiVersion": "external-secrets.io/v1",
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
