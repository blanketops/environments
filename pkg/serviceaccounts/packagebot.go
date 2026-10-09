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
	"maps"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"
	intent "github.com/blanketops/environments/pkg/intent/package"
	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

// PackageServiceAccountReconciler converges the ServiceAccount a Package's
// kapp App deploys as.
type PackageServiceAccountReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewPackageServiceAccountReconciler constructs a
// PackageServiceAccountReconciler.
func NewPackageServiceAccountReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
) *PackageServiceAccountReconciler {
	return &PackageServiceAccountReconciler{
		Client: c,
		Scheme: scheme,
		Log:    log,
	}
}

// Reconcile ensures the ServiceAccount exists for the Package's kapp App.
//
// RULES:
// - ServiceAccount is ALWAYS created
// - Its name is derived from the Package, the same way the App names it
// - It grants nothing by itself; what it may deploy is bound to it separately
func (r *PackageServiceAccountReconciler) Reconcile(
	ctx context.Context,
	pkg *packageResolution.ResolvedPackage,
) error {

	if pkg == nil || pkg.Package == nil || pkg.Spec == nil {
		return nil
	}

	saName := packageServiceAccountName(pkg)
	namespace := pkg.Package.Namespace
	labels := intent.BlanketOpsLabels(pkg.Package.Labels)

	desired := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					pkg.Package,
					environmentv1alpha1.GroupVersion.WithKind("Package"),
				),
			},
		},
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
				"Creating ServiceAccount for Package",
				"package", pkg.Package.Name,
				"serviceAccount", saName,
			)
			return r.Client.Create(ctx, desired)
		}
		return err
	}

	// ---------------------------------------------------------------------
	// Reconcile labels (ONLY mutable field)
	// ---------------------------------------------------------------------

	changed := false
	for k, v := range labels {
		if existing.Labels[k] != v {
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	maps.Copy(existing.Labels, labels)

	r.Log.Info(
		"Updating ServiceAccount labels",
		"serviceAccount", saName,
	)

	return r.Client.Update(ctx, existing)
}

// Delete removes the ServiceAccount created for the Package's kapp App.
// Mirrors the name-resolution logic in Reconcile so it targets the same
// object. Idempotent — a missing ServiceAccount is not an error.
func (r *PackageServiceAccountReconciler) Delete(
	ctx context.Context,
	pkg *packageResolution.ResolvedPackage,
) error {
	if pkg == nil || pkg.Package == nil || pkg.Spec == nil {
		return nil
	}

	saName := packageServiceAccountName(pkg)

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: pkg.Package.Namespace,
		},
	}

	r.Log.Info(
		"Deleting ServiceAccount for Package",
		"package", pkg.Package.Name,
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

// packageServiceAccountName is the name of the ServiceAccount for a resolved
// Package. The name itself is decided in one place, the domain.
func packageServiceAccountName(pkg *packageResolution.ResolvedPackage) string {
	return domain.PackageID{
		Namespace: pkg.Package.Namespace,
		Name:      pkg.Package.Name,
	}.ServiceAccountName()
}
