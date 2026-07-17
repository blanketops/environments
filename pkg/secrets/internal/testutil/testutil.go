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
Package testutil provides fake-client test helpers shared by every
reconciler package under pkg/secrets — scheme construction and
ExternalSecret/Secret assertions. It lives under internal/ so it is only
importable from within pkg/secrets, where it is exclusively test-only code.
*/
package testutil

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewScheme builds a runtime.Scheme with corev1 registered, plus any
// additional AddToScheme functions the caller needs — typically the
// environments-api CR groups required for controllerutil.SetControllerReference.
func NewScheme(t *testing.T, adders ...func(*runtime.Scheme) error) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	for _, add := range adders {
		if err := add(scheme); err != nil {
			t.Fatalf("add to scheme: %v", err)
		}
	}
	return scheme
}

// GetExternalSecret fetches the named ExternalSecret, failing the test if
// it does not exist.
func GetExternalSecret(t *testing.T, c client.Client, name, namespace string) *unstructured.Unstructured {
	t.Helper()
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("external-secrets.io/v1")
	u.SetKind("ExternalSecret")
	if err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, u); err != nil {
		t.Fatalf("get ExternalSecret %s/%s: %v", namespace, name, err)
	}
	return u
}

// ExternalSecretExists reports whether the named ExternalSecret exists.
func ExternalSecretExists(t *testing.T, c client.Client, name, namespace string) bool {
	t.Helper()
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("external-secrets.io/v1")
	u.SetKind("ExternalSecret")
	err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, u)
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("get ExternalSecret %s/%s: %v", namespace, name, err)
	return false
}

// SecretExists reports whether the named Secret exists.
func SecretExists(t *testing.T, c client.Client, name, namespace string) bool {
	t.Helper()
	s := &corev1.Secret{}
	err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, s)
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("get Secret %s/%s: %v", namespace, name, err)
	return false
}
