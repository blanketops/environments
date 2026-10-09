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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

const (
	deployerRole    = "package-deployer-role"
	deployerBinding = "blanketops-package-default-app"
)

func getDeployerBinding(t *testing.T, c client.Client) *rbacv1.ClusterRoleBinding {
	t.Helper()
	b := &rbacv1.ClusterRoleBinding{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: deployerBinding}, b); err != nil {
		t.Fatalf("get ClusterRoleBinding: %v", err)
	}
	return b
}

func assertBindsPackageAccount(t *testing.T, b *rbacv1.ClusterRoleBinding) {
	t.Helper()
	wantRole := rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: deployerRole}
	if b.RoleRef != wantRole {
		t.Errorf("roleRef = %+v, want %+v", b.RoleRef, wantRole)
	}
	wantSubject := rbacv1.Subject{Kind: "ServiceAccount", Name: packageSAName, Namespace: "default"}
	if len(b.Subjects) != 1 || b.Subjects[0] != wantSubject {
		t.Errorf("subjects = %+v, want only %+v", b.Subjects, wantSubject)
	}
}

func TestPackageDeployerBindingReconciler_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)

	for name, pkg := range map[string]*packageResolution.ResolvedPackage{
		"nil":         nil,
		"nil Package": {Spec: &packageResolution.ResolvedPackageSpec{}},
		"nil Spec":    {Package: &environmentv1alpha1.Package{}},
	} {
		if err := r.Reconcile(context.Background(), pkg); err != nil {
			t.Errorf("Reconcile(%s) should no-op, got: %v", name, err)
		}
		if err := r.Delete(context.Background(), pkg); err != nil {
			t.Errorf("Delete(%s) should no-op, got: %v", name, err)
		}
	}
}

// Without a role to bind there is nothing safe to create.
func TestPackageDeployerBindingReconciler_Reconcile_NoClusterRole(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), "")

	if err := r.Reconcile(context.Background(), newResolvedPackage(nil)); err == nil {
		t.Fatal("Reconcile with no cluster role = nil error, want error")
	}
	list := &rbacv1.ClusterRoleBindingList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("nothing should be created, found %d ClusterRoleBindings", len(list.Items))
	}
}

func TestPackageDeployerBindingReconciler_Reconcile_Creates(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)
	pkg := newResolvedPackage(map[string]string{
		"environments.blanketops.dev/name": "app",
		"unrelated":                        "x",
	})

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	b := getDeployerBinding(t, c)
	assertBindsPackageAccount(t, b)
	if len(b.Labels) != 1 || b.Labels["environments.blanketops.dev/name"] != "app" {
		t.Errorf("labels = %v, want only the environments.blanketops.dev one", b.Labels)
	}
	if len(b.OwnerReferences) != 0 {
		t.Errorf("ownerReferences = %v, a cluster-scoped object cannot be owned by the Package", b.OwnerReferences)
	}
}

// A second Reconcile leaves the same object and writes nothing.
func TestPackageDeployerBindingReconciler_Reconcile_Idempotent(t *testing.T) {
	writes := 0
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			writes++
			return c.Update(ctx, obj, opts...)
		},
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			writes++
			return c.Delete(ctx, obj, opts...)
		},
	}).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)
	pkg := newResolvedPackage(map[string]string{"environments.blanketops.dev/name": "app"})

	for i := range 2 {
		if err := r.Reconcile(context.Background(), pkg); err != nil {
			t.Fatalf("Reconcile %d: %v", i+1, err)
		}
	}

	list := &rbacv1.ClusterRoleBindingList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 || writes != 0 {
		t.Errorf("ClusterRoleBindings = %d writes = %d, want 1 and 0", len(list.Items), writes)
	}
}

// Extra subjects are removed: the binding grants the role to the Package's
// ServiceAccount and to nobody else.
func TestPackageDeployerBindingReconciler_Reconcile_RestoresSubjectsAndLabels(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithObjects(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: deployerBinding},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: packageSAName, Namespace: "default"},
			{Kind: "ServiceAccount", Name: "someone-else", Namespace: "other"},
		},
		RoleRef: rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: deployerRole},
	}).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)

	if err := r.Reconcile(context.Background(), newResolvedPackage(map[string]string{"environments.blanketops.dev/name": "app"})); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	b := getDeployerBinding(t, c)
	assertBindsPackageAccount(t, b)
	if b.Labels["environments.blanketops.dev/name"] != "app" {
		t.Errorf("labels = %v, want the Package's name label", b.Labels)
	}
}

// roleRef cannot be updated, so a binding to another role is replaced.
func TestPackageDeployerBindingReconciler_Reconcile_ReplacesOtherRole(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithObjects(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: deployerBinding},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: packageSAName, Namespace: "default"}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cluster-admin"},
	}).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)

	if err := r.Reconcile(context.Background(), newResolvedPackage(nil)); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	assertBindsPackageAccount(t, getDeployerBinding(t, c))
}

func TestPackageDeployerBindingReconciler_Reconcile_Errors(t *testing.T) {
	boom := errors.New("boom")
	otherRole := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: deployerBinding},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cluster-admin"},
	}

	t.Run("get", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return boom
			},
		}).Build()
		r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)
		if err := r.Reconcile(context.Background(), newResolvedPackage(nil)); !errors.Is(err, boom) {
			t.Fatalf("Reconcile error = %v, want the Get error", err)
		}
	})

	t.Run("delete of a binding to another role", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithObjects(otherRole.DeepCopy()).WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return boom
			},
		}).Build()
		r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)
		if err := r.Reconcile(context.Background(), newResolvedPackage(nil)); !errors.Is(err, boom) {
			t.Fatalf("Reconcile error = %v, want the Delete error", err)
		}
	})
}

func TestPackageDeployerBindingReconciler_Delete(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)
	pkg := newResolvedPackage(nil)

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	// Deleting twice: the second finds nothing and is not an error.
	for i := range 2 {
		if err := r.Delete(context.Background(), pkg); err != nil {
			t.Fatalf("Delete %d: %v", i+1, err)
		}
	}

	err := c.Get(context.Background(), types.NamespacedName{Name: deployerBinding}, &rbacv1.ClusterRoleBinding{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("ClusterRoleBinding should be gone, got: %v", err)
	}
}

func TestPackageDeployerBindingReconciler_Delete_Error(t *testing.T) {
	boom := errors.New("boom")
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
			return boom
		},
	}).Build()
	r := NewPackageDeployerBindingReconciler(c, logr.Discard(), deployerRole)

	if err := r.Delete(context.Background(), newResolvedPackage(nil)); !errors.Is(err, boom) {
		t.Fatalf("Delete error = %v, want the client error", err)
	}
}
