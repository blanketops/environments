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
Package hookurl reconciles the webhook hook-url secret — a plain Secret (no
ESO involvement, since the URL is user-declared on the GitRepository CR
rather than sourced from an external store) that exists only because
Upjet's RepositoryWebhook consumes the URL via urlSecretRef.
*/
package hookurl

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gitrepoResolution "github.com/blanketops/environments/resolution/gitrepository/resolve"
)

// HookURLSecretKey is the key under which the webhook URL is stored in the
// reconciled Secret's data.
const HookURLSecretKey = "url"

// HookURLSecretName returns the per-repository hookurl Secret name.
func HookURLSecretName(repoName string) string {
	return repoName + "-hookurl"
}

// HookURLSecretReconciler materializes the webhook delivery URL as a plain
// Secret. The URL's source of truth is spec.contract.hookUrl on the
// GitRepository CR — user-declared, not store-managed. No ESO involvement:
// there is no external producer for this value, so an ExternalSecret would
// reference a key nothing populates. The Secret exists only because Upjet's
// RepositoryWebhook consumes the URL via urlSecretRef (the Terraform provider
// marks webhook URLs sensitive; Upjet mechanically converts sensitive fields
// to secret references).
type HookURLSecretReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewHookURLSecretReconciler constructs a HookURLSecretReconciler.
func NewHookURLSecretReconciler(c client.Client, scheme *runtime.Scheme, log logr.Logger) *HookURLSecretReconciler {
	return &HookURLSecretReconciler{Client: c, Scheme: scheme, Log: log}
}

// Reconcile creates or updates the hookurl Secret for resolved's repository
// from its contract-declared hookUrl.
func (r *HookURLSecretReconciler) Reconcile(ctx context.Context, resolved *gitrepoResolution.ResolvedGitRepository) error {
	if resolved == nil || resolved.Repository == nil || resolved.Spec == nil {
		return fmt.Errorf("resolved gitrepository is incomplete")
	}
	repo := resolved.Repository
	hookURL := resolved.Spec.HookURL
	if hookURL == "" {
		// Resolution validates this as required — reaching here is a resolver bug.
		return fmt.Errorf("gitrepository %s/%s: empty hookUrl (resolver bug)", repo.Namespace, repo.Name)
	}

	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HookURLSecretName(repo.Name),
			Namespace: repo.Namespace,
			Labels: map[string]string{
				"sources.blanketops.dev/gitrepository": repo.Name,
				"blanketops.dev/managed":               "true",
				"blanketops.dev/purpose":               "webhook",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{HookURLSecretKey: hookURL},
	}
	if err := controllerutil.SetControllerReference(repo, desired, r.Scheme); err != nil {
		return err
	}

	var existing corev1.Secret
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(desired), &existing)
	if apierrors.IsNotFound(err) {
		r.Log.Info("creating hookurl secret", "repository", repo.Name, "secret", desired.Name)
		return r.Client.Create(ctx, desired)
	}
	if err != nil {
		return err
	}
	if string(existing.Data[HookURLSecretKey]) != hookURL {
		existing.StringData = map[string]string{HookURLSecretKey: hookURL}
		r.Log.Info("updating hookurl secret", "repository", repo.Name, "secret", desired.Name)
		return r.Client.Update(ctx, &existing)
	}
	r.Log.V(1).Info("hookurl secret up-to-date", "repository", repo.Name)
	return nil
}

// Delete removes repo's hookurl Secret, if it exists.
func (r *HookURLSecretReconciler) Delete(ctx context.Context, repo *gitrepoResolution.ResolvedGitRepository) error {
	if repo == nil {
		return nil
	}
	obj := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: HookURLSecretName(repo.Repository.Name), Namespace: repo.Repository.Namespace,
	}}
	if err := r.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
