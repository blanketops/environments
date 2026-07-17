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

package fluxcd

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	deploymentResolution "github.com/blanketops/environments/resolution/deployment/resolve"

	"github.com/blanketops/environments/pkg/secrets/internal/testutil"
)

func newResolvedDeploymentForFlux(name, namespace string) *deploymentResolution.ResolvedDeployment {
	return &deploymentResolution.ResolvedDeployment{
		Deployment: &environmentv1alpha1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: "deployment-uid"},
		},
	}
}

func TestGenerateSSHKeypair_RoundTrips(t *testing.T) {
	privPEM, pubAuthorized, err := generateSSHKeypair("test-comment")
	if err != nil {
		t.Fatalf("generateSSHKeypair: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatalf("parse generated private key: %v", err)
	}

	wantPub := string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
	if string(pubAuthorized) != wantPub {
		t.Fatalf("public key does not match private key: got %q want %q", pubAuthorized, wantPub)
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_Reconcile_GeneratesNewKeypair(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if err := r.Reconcile(context.Background(), depl); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-deployment-flux-ssh", Namespace: "default"}, secret); err != nil {
		t.Fatalf("get flux ssh secret: %v", err)
	}
	if len(secret.Data["identity"]) == 0 || len(secret.Data["identity.pub"]) == 0 {
		t.Fatalf("expected identity and identity.pub to be populated, got: %+v", secret.Data)
	}
	if string(secret.Data["known_hosts"]) != githubKnownHosts {
		t.Fatalf("expected known_hosts to be the pinned GitHub host key")
	}
	owners := secret.GetOwnerReferences()
	if len(owners) != 1 || owners[0].Name != "my-deployment" {
		t.Fatalf("expected owner reference to Deployment, got %+v", owners)
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_Reconcile_SkipsWhenKeyExists(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if err := r.Reconcile(context.Background(), depl); err != nil {
		t.Fatalf("initial Reconcile: %v", err)
	}
	before := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-deployment-flux-ssh", Namespace: "default"}, before); err != nil {
		t.Fatalf("get flux ssh secret: %v", err)
	}

	if err := r.Reconcile(context.Background(), depl); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	after := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "my-deployment-flux-ssh", Namespace: "default"}, after); err != nil {
		t.Fatalf("get flux ssh secret: %v", err)
	}

	if string(before.Data["identity"]) != string(after.Data["identity"]) {
		t.Fatal("expected key to remain stable across reconciles, but it was rotated")
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_PublicKey_ReturnsStoredPublicKey(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if err := r.Reconcile(context.Background(), depl); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	pub, err := r.PublicKey(context.Background(), depl)
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if pub == "" {
		t.Fatal("expected non-empty public key")
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_PublicKey_DerivesFromPrivateKeyWhenMissing(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	privPEM, pubAuthorized, err := generateSSHKeypair("derive-test")
	if err != nil {
		t.Fatalf("generateSSHKeypair: %v", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deployment-flux-ssh", Namespace: "default"},
		Data: map[string][]byte{
			"identity": privPEM,
			// identity.pub intentionally omitted to exercise the fallback path.
		},
	}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatalf("seed secret: %v", err)
	}

	pub, err := r.PublicKey(context.Background(), depl)
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	if pub != string(pubAuthorized) {
		t.Fatalf("expected derived public key to match generated one: got %q want %q", pub, pubAuthorized)
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_PublicKey_ErrorsWhenSecretMissing(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if _, err := r.PublicKey(context.Background(), depl); err == nil {
		t.Fatal("expected error when flux ssh secret does not exist")
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_Delete_RemovesSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if err := r.Reconcile(context.Background(), depl); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if err := r.Delete(context.Background(), depl); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	err := c.Get(context.Background(), client.ObjectKey{Name: "my-deployment-flux-ssh", Namespace: "default"}, &corev1.Secret{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected secret to be deleted, got err: %v", err)
	}
}

func TestDeploymentFluxGitSSHSecretReconciler_Delete_IdempotentWhenNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testutil.NewScheme(t, environmentv1alpha1.AddToScheme)).Build()
	r := NewDeploymentFluxGitSSHSecretReconciler(c, logr.Discard())
	depl := newResolvedDeploymentForFlux("my-deployment", "default")

	if err := r.Delete(context.Background(), depl); err != nil {
		t.Fatalf("expected Delete on nonexistent secret to be a no-op, got: %v", err)
	}
}
