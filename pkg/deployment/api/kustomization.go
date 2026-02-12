package api

import (
	"context"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KustomizationReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func New(c client.Client, scheme *runtime.Scheme, log logr.Logger) *KustomizationReconciler {
	return &KustomizationReconciler{Client: c, Scheme: scheme, Log: log}
}

// reconcileKustomization creates Kustomization CR that applies the manifests
func (m *KustomizationReconciler) ReconcileKustomization(ctx context.Context, env *environmentv1.Environment, clusterPath string) error {
	// kust := &kustomizev1.Kustomization{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      fmt.Sprintf("%s-kustomize", env.Spec.ApplicationName),
	// 		Namespace: env.Namespace,
	// 	},
	// 	Spec: kustomizev1.KustomizationSpec{
	// 		Interval: metav1.Duration{Duration: 1 * time.Minute},
	// 		SourceRef: kustomizev1.CrossNamespaceSourceReference{
	// 			Kind: "GitRepository",
	// 			Name: fmt.Sprintf("%s-source", env.Spec.Deploy.Name),
	// 		},
	// 		Path: clusterPath,
	// 	},
	// }

	// if err := controllerutil.SetControllerReference(env, kust, m.Scheme); err != nil {
	// 	return err
	// }

	// m.Log.Info("Applying Kustomization", "name", kust.Name)
	// return utils.CreateOrPatch(ctx, m.Client, kust)
	return nil
}
