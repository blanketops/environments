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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	deploymentResolution "github.com/blanketops/environments/resolution/deployment"
)

type DeploymentGitSSHSecretReconciler struct {
	Client    client.Client
	Log       logr.Logger
	StoreName string
	StoreKind string
}

func NewDeploymentGitSSHSecretReconciler(c client.Client, log logr.Logger, storeName string, storeKind string) *DeploymentGitSSHSecretReconciler {
	return &DeploymentGitSSHSecretReconciler{
		Client:    c,
		Log:       log,
		StoreName: storeName,
		StoreKind: storeKind,
	}
}

func (r *DeploymentGitSSHSecretReconciler) Reconcile(ctx context.Context, deployment *deploymentResolution.ResolvedDeployment) error {
	source := deployment.Spec.ManifestsRepo
	secretName := source.CloneSecret
	namespace := deployment.Deployment.Namespace

	desired := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ExternalSecret",
			"metadata": map[string]any{
				"name":      secretName,
				"namespace": namespace,
				"labels": map[string]any{
					"blanketops.dev/managed":    "true",
					"blanketops.dev/purpose":    "git-ssh",
					"blanketops.dev/deployment": deployment.Deployment.Name,
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

	if err := controllerutil.SetControllerReference(
		deployment.Deployment,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: namespace}, &existing)

	if apierrors.IsNotFound(err) {
		r.Log.Info("creating ExternalSecret for Git SSH",
			"deployment", deployment.Deployment.Name,
			"secret", secretName,
			"store", r.StoreName,
		)
		return r.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(existing.Object["spec"], desired.Object["spec"]) {
		existing.Object["spec"] = desired.Object["spec"]
		r.Log.Info("updating ExternalSecret for Git SSH",
			"deployment", deployment.Deployment.Name,
			"secret", secretName,
			"store", r.StoreName,
		)
		return r.Client.Update(ctx, &existing)
	}

	r.Log.V(1).Info("ExternalSecret already up-to-date",
		"deployment", deployment.Deployment.Name,
		"secret", secretName,
	)
	return nil
}

// git/deployment.go — append
func (r *DeploymentGitSSHSecretReconciler) Delete(ctx context.Context, deployment *deploymentResolution.ResolvedDeployment) error {
	secretName := deployment.Spec.ManifestsRepo.CloneSecret
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("external-secrets.io/v1")
	obj.SetKind("ExternalSecret")
	obj.SetName(secretName)
	obj.SetNamespace(deployment.Deployment.Namespace)
	if err := r.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
