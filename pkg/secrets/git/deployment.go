package git

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	deploymentResolution "github.com/ntlaletsi70/blanketops-environments-mvp/internal/resolution/deployment"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DeploymentGitSSHSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewDeploymentGitSSHSecretReconciler(
	c client.Client,
	log logr.Logger,
) *DeploymentGitSSHSecretReconciler {
	return &DeploymentGitSSHSecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *DeploymentGitSSHSecretReconciler) Reconcile(
	ctx context.Context,
	deployment *deploymentResolution.ResolvedDeployment,
) error {

	source := deployment.Spec.ManifestsRepo

	secretName := source.CloneSecret
	namespace := deployment.Deployment.Namespace

	// -------------------------------------------------------------------------
	// Desired ExternalSecret (UNSTRUCTURED)
	// -------------------------------------------------------------------------
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
				"refreshInterval": "0s",
				"secretStoreRef": map[string]any{
					"name": "blanketops-environments-fake",
					"kind": "ClusterSecretStore",
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

	// -------------------------------------------------------------------------
	// Ownership (Deployment → ExternalSecret)
	// -------------------------------------------------------------------------
	if err := controllerutil.SetControllerReference(
		deployment.Deployment,
		desired,
		r.Client.Scheme(),
	); err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Fetch existing
	// -------------------------------------------------------------------------
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      secretName,
			Namespace: namespace,
		},
		&existing,
	)

	// -------------------------------------------------------------------------
	// Create
	// -------------------------------------------------------------------------
	if apierrors.IsNotFound(err) {
		r.Log.Info(
			"Creating ExternalSecret for Git SSH",
			"Deployment", deployment.Deployment.Name,
			"secret", secretName,
		)
		return r.Client.Create(ctx, desired)
	}

	if err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// Update (spec drift only)
	// -------------------------------------------------------------------------
	if !reflect.DeepEqual(
		existing.Object["spec"],
		desired.Object["spec"],
	) {
		existing.Object["spec"] = desired.Object["spec"]

		r.Log.Info(
			"Updating ExternalSecret for Git SSH",
			"Deployment", deployment.Deployment.Name,
			"secret", secretName,
		)

		return r.Client.Update(ctx, &existing)
	}

	// -------------------------------------------------------------------------
	// No-op
	// -------------------------------------------------------------------------
	r.Log.V(1).Info(
		"ExternalSecret already up-to-date",
		"Deployment", deployment.Deployment.Name,
		"secret", secretName,
	)

	return nil
}
