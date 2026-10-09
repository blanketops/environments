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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	packageResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

const packageSAName = "app-package"

func newPackageSAScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(core): %v", err)
	}
	if err := environmentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(environments): %v", err)
	}
	return scheme
}

func newResolvedPackage(labels map[string]string) *packageResolution.ResolvedPackage {
	return &packageResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", UID: "package-uid", Labels: labels},
		},
		Spec: &packageResolution.ResolvedPackageSpec{},
	}
}

func getPackageSA(t *testing.T, c client.Client) *corev1.ServiceAccount {
	t.Helper()
	sa := &corev1.ServiceAccount{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: packageSAName}, sa); err != nil {
		t.Fatalf("get ServiceAccount: %v", err)
	}
	return sa
}

func TestPackageServiceAccountReconciler_NilGuards(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())

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

	list := &corev1.ServiceAccountList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("nothing should be created, found %d ServiceAccounts", len(list.Items))
	}
}

func TestPackageServiceAccountReconciler_Reconcile_CreatesOwnedAndLabelled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())
	pkg := newResolvedPackage(map[string]string{
		"environments.blanketops.dev/name": "app",
		"environments.blanketops.dev/type": "dev",
		"unrelated":                        "x",
	})

	if err := r.Reconcile(context.Background(), pkg); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	sa := getPackageSA(t, c)
	if len(sa.Labels) != 2 || sa.Labels["environments.blanketops.dev/name"] != "app" || sa.Labels["environments.blanketops.dev/type"] != "dev" {
		t.Errorf("labels = %v, want only the two environments.blanketops.dev ones", sa.Labels)
	}
	if len(sa.OwnerReferences) != 1 {
		t.Fatalf("ownerReferences = %v, want one", sa.OwnerReferences)
	}
	owner := sa.OwnerReferences[0]
	if owner.Kind != "Package" || owner.APIVersion != environmentv1alpha1.GroupVersion.String() ||
		owner.Name != "app" || owner.UID != "package-uid" || owner.Controller == nil || !*owner.Controller {
		t.Errorf("owner = %+v, want the Package as controller", owner)
	}
}

// A second Reconcile leaves the same object and writes nothing.
func TestPackageServiceAccountReconciler_Reconcile_Idempotent(t *testing.T) {
	updates := 0
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			updates++
			return c.Update(ctx, obj, opts...)
		},
	}).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())
	pkg := newResolvedPackage(map[string]string{"environments.blanketops.dev/name": "app"})

	for i := range 2 {
		if err := r.Reconcile(context.Background(), pkg); err != nil {
			t.Fatalf("Reconcile %d: %v", i+1, err)
		}
	}

	list := &corev1.ServiceAccountList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 || updates != 0 {
		t.Errorf("ServiceAccounts = %d updates = %d, want 1 and 0", len(list.Items), updates)
	}
}

// An existing ServiceAccount gets the Package's labels and keeps its own.
func TestPackageServiceAccountReconciler_Reconcile_UpdatesLabels(t *testing.T) {
	tests := map[string]map[string]string{
		"no labels yet":   nil,
		"stale and other": {"environments.blanketops.dev/name": "old", "kept": "yes"},
	}
	for name, existing := range tests {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithObjects(&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: packageSAName, Namespace: "default", Labels: existing},
			}).Build()
			r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())

			if err := r.Reconcile(context.Background(), newResolvedPackage(map[string]string{"environments.blanketops.dev/name": "app"})); err != nil {
				t.Fatalf("Reconcile: %v", err)
			}

			sa := getPackageSA(t, c)
			if sa.Labels["environments.blanketops.dev/name"] != "app" {
				t.Errorf("labels = %v, want the Package's name label", sa.Labels)
			}
			if existing["kept"] != sa.Labels["kept"] {
				t.Errorf("labels = %v, an unrelated label was dropped", sa.Labels)
			}
		})
	}
}

func TestPackageServiceAccountReconciler_Reconcile_GetError(t *testing.T) {
	boom := errors.New("boom")
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return boom
		},
	}).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())

	if err := r.Reconcile(context.Background(), newResolvedPackage(nil)); !errors.Is(err, boom) {
		t.Fatalf("Reconcile error = %v, want the Get error", err)
	}
}

func TestPackageServiceAccountReconciler_Delete(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())
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

	err := c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: packageSAName}, &corev1.ServiceAccount{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("ServiceAccount should be gone, got: %v", err)
	}
}

func TestPackageServiceAccountReconciler_Delete_Error(t *testing.T) {
	boom := errors.New("boom")
	c := fake.NewClientBuilder().WithScheme(newPackageSAScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
			return boom
		},
	}).Build()
	r := NewPackageServiceAccountReconciler(c, c.Scheme(), logr.Discard())

	if err := r.Delete(context.Background(), newResolvedPackage(nil)); !errors.Is(err, boom) {
		t.Fatalf("Delete error = %v, want the client error", err)
	}
}
