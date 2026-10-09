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
	"fmt"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intent "github.com/blanketops/environments/pkg/intent/package"
	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

// PackageDeployerBindingReconciler converges the ClusterRoleBinding that
// grants a Package's ServiceAccount what its kapp App may deploy.
//
// The rights themselves are not decided here. They are one ClusterRole,
// shipped with the installation, and this binds each Package's
// ServiceAccount to it.
type PackageDeployerBindingReconciler struct {
	Client client.Client
	Log    logr.Logger
	// ClusterRole is the name of the ClusterRole to bind.
	ClusterRole string
}

// NewPackageDeployerBindingReconciler constructs a
// PackageDeployerBindingReconciler binding to the named ClusterRole.
func NewPackageDeployerBindingReconciler(
	c client.Client,
	log logr.Logger,
	clusterRole string,
) *PackageDeployerBindingReconciler {
	return &PackageDeployerBindingReconciler{
		Client:      c,
		Log:         log,
		ClusterRole: clusterRole,
	}
}

// Reconcile ensures the ClusterRoleBinding exists for the Package's
// ServiceAccount.
//
// RULES:
// - The binding has exactly one subject, the Package's ServiceAccount
// - It is cluster-scoped, so the Package cannot own it; Delete removes it
// - A binding to another role is replaced: roleRef cannot be updated
func (r *PackageDeployerBindingReconciler) Reconcile(
	ctx context.Context,
	pkg *packageResolution.ResolvedPackage,
) error {

	if pkg == nil || pkg.Package == nil || pkg.Spec == nil {
		return nil
	}
	if r.ClusterRole == "" {
		return fmt.Errorf("package deployer cluster role is not configured")
	}

	id := packageID(pkg)
	name := id.DeployerBindingName()

	desired := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: intent.BlanketOpsLabels(pkg.Package.Labels),
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      id.ServiceAccountName(),
			Namespace: id.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     r.ClusterRole,
		},
	}

	// ---------------------------------------------------------------------
	// Fetch existing ClusterRoleBinding
	// ---------------------------------------------------------------------

	existing := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: name}, existing)

	// ---------------------------------------------------------------------
	// Create if missing
	// ---------------------------------------------------------------------

	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info(
				"Creating ClusterRoleBinding for Package",
				"package", id.Name,
				"clusterRoleBinding", name,
				"clusterRole", r.ClusterRole,
			)
			return r.Client.Create(ctx, desired)
		}
		return err
	}

	// ---------------------------------------------------------------------
	// Replace when bound to another role (roleRef is immutable)
	// ---------------------------------------------------------------------

	if existing.RoleRef != desired.RoleRef {
		r.Log.Info(
			"Replacing ClusterRoleBinding bound to another role",
			"clusterRoleBinding", name,
			"clusterRole", r.ClusterRole,
		)
		if err := r.Client.Delete(ctx, existing); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		return r.Client.Create(ctx, desired)
	}

	// ---------------------------------------------------------------------
	// Reconcile subjects and labels
	// ---------------------------------------------------------------------

	if apiequality.Semantic.DeepEqual(existing.Subjects, desired.Subjects) &&
		apiequality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		return nil
	}

	existing.Subjects = desired.Subjects
	existing.Labels = desired.Labels

	r.Log.Info(
		"Updating ClusterRoleBinding for Package",
		"clusterRoleBinding", name,
	)

	return r.Client.Update(ctx, existing)
}

// Delete removes the ClusterRoleBinding created for the Package. Idempotent —
// a missing ClusterRoleBinding is not an error.
func (r *PackageDeployerBindingReconciler) Delete(
	ctx context.Context,
	pkg *packageResolution.ResolvedPackage,
) error {
	if pkg == nil || pkg.Package == nil || pkg.Spec == nil {
		return nil
	}

	id := packageID(pkg)
	name := id.DeployerBindingName()

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	r.Log.Info(
		"Deleting ClusterRoleBinding for Package",
		"package", id.Name,
		"clusterRoleBinding", name,
	)

	if err := r.Client.Delete(ctx, binding); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}
