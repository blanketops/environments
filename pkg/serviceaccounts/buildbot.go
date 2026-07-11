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

package serviceaccounts

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildResolution "github.com/BlanketOps/environments/resolution/build"
)

type ServiceAccountReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewServiceAccountReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
) *ServiceAccountReconciler {
	return &ServiceAccountReconciler{
		Client: c,
		Scheme: scheme,
		Log:    log,
	}
}

// Reconcile ensures a ServiceAccount exists for the Build execution.
//
// RULES:
// - ServiceAccount is ALWAYS created
// - Contract values are honored when provided
// - Defaults are generated when omitted
func (r *ServiceAccountReconciler) Reconcile(
	ctx context.Context,
	build *buildResolution.ResolvedBuild,
) error {

	if build == nil || build.Build == nil || build.Spec == nil {
		return nil
	}

	spec := build.Spec
	namespace := build.Build.Namespace

	// ---------------------------------------------------------------------
	// Resolve desired ServiceAccount identity (NO re-typing)
	// ---------------------------------------------------------------------

	saName := build.Build.Name + "-build"
	secretName := ""

	if spec.ServiceAccount != nil {
		if spec.ServiceAccount.Name != "" {
			saName = spec.ServiceAccount.Name
		}
		secretName = spec.ServiceAccount.Secret
	}

	desired := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					build.Build,
					build.Build.GroupVersionKind(),
				),
			},
		},
	}

	// Attach image pull secret if requested
	if secretName != "" {
		desired.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: secretName},
		}
	}

	// ---------------------------------------------------------------------
	// Fetch existing ServiceAccount
	// ---------------------------------------------------------------------

	existing := &corev1.ServiceAccount{}
	err := r.Client.Get(
		ctx,
		types.NamespacedName{
			Name:      saName,
			Namespace: namespace,
		},
		existing,
	)

	// ---------------------------------------------------------------------
	// Create if missing
	// ---------------------------------------------------------------------

	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info(
				"Creating ServiceAccount for Build",
				"build", build.Build.Name,
				"serviceAccount", saName,
			)
			return r.Client.Create(ctx, desired)
		}
		return err
	}

	// ---------------------------------------------------------------------
	// Reconcile imagePullSecrets (ONLY mutable field)
	// ---------------------------------------------------------------------

	if secretName != "" {
		if len(existing.ImagePullSecrets) != 1 ||
			existing.ImagePullSecrets[0].Name != secretName {

			existing.ImagePullSecrets = desired.ImagePullSecrets

			r.Log.Info(
				"Updating ServiceAccount imagePullSecrets",
				"serviceAccount", saName,
				"secret", secretName,
			)

			return r.Client.Update(ctx, existing)
		}
	}

	return nil
}

// Delete removes the ServiceAccount created for the Build execution.
// Mirrors the name-resolution logic in Reconcile so it targets the same
// object. Idempotent — a missing ServiceAccount is not an error.
func (r *ServiceAccountReconciler) Delete(
	ctx context.Context,
	build *buildResolution.ResolvedBuild,
) error {
	if build == nil || build.Build == nil || build.Spec == nil {
		return nil
	}

	spec := build.Spec
	namespace := build.Build.Namespace

	saName := build.Build.Name + "-build"
	if spec.ServiceAccount != nil && spec.ServiceAccount.Name != "" {
		saName = spec.ServiceAccount.Name
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
		},
	}

	r.Log.Info(
		"Deleting ServiceAccount for Build",
		"build", build.Build.Name,
		"serviceAccount", saName,
	)

	if err := r.Client.Delete(ctx, sa); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}
