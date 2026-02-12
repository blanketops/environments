package api

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KnativeReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewKnativeReconciler(c client.Client, scheme *runtime.Scheme, log logr.Logger) *KnativeReconciler {
	return &KnativeReconciler{Client: c, Scheme: scheme, Log: log}
}

// func (r *KnativeReconciler) ReconcileKnative(ctx context.Context, env *environmentv1.Environment) (bool, error) {
// 	for _, su := range env.Spec.ServiceUnits {
// 		// Placeholder: override image if built from Buildpacks
// 		if su.ImageFromBuild && env.Spec.Build.Image != "" {
// 			su.Image = env.Spec.Build.Image
// 		}

// 		// Create the Knative Service object
// 		svc := &servingv1.Service{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      su.Name,
// 				Namespace: env.Namespace,
// 				Labels: map[string]string{
// 					"environment": env.Name,
// 					"serviceUnit": su.Name,
// 				},
// 			},
// 			Spec: servingv1.ServiceSpec{
// 				ConfigurationSpec: servingv1.ConfigurationSpec{
// 					Template: servingv1.RevisionTemplateSpec{
// 						Spec: servingv1.RevisionSpec{
// 							PodSpec: envToKnativePodSpec(su),
// 						},
// 					},
// 				},
// 			},
// 		}

// 		if err := controllerutil.SetControllerReference(env, svc, r.Scheme); err != nil {
// 			return false, fmt.Errorf("setting owner ref on knative service: %w", err)
// 		}

// 		r.Log.Info("Applying Knative Service", "name", su.Name)
// 		if err := r.Client.Patch(ctx, svc, client.Apply, &client.PatchOptions{
// 			FieldManager: "knative-manager",
// 			Force:        pointerBool(true),
// 		}); err != nil {
// 			return false, fmt.Errorf("patching knative service: %w", err)
// 		}

// 		// Optional: wait for readiness (simplified)
// 		ready, err := r.isServiceReady(ctx, svc)
// 		if err != nil {
// 			return false, err
// 		}
// 		if !ready {
// 			r.Log.Info("Knative Service not yet ready, requeueing", "service", su.Name)
// 			time.Sleep(5 * time.Second) // simple retry placeholder
// 			return false, nil
// 		}
// 	}

// 	return true, nil
// }

// // Helper: translate ServiceUnit to Knative PodSpec
// // envToKnativePodSpec converts a ServiceUnit to corev1.PodSpec for Knative Service
// func envToKnativePodSpec(su environmentv1.ServiceUnit) corev1.PodSpec {
// 	return corev1.PodSpec{
// 		Containers: []corev1.Container{
// 			{
// 				Name:  su.Name,
// 				Image: su.Image,
// 				Ports: []corev1.ContainerPort{
// 					{ContainerPort: su.ContainerPort},
// 				},
// 			},
// 		},
// 	}
// }

// // Helper: simple readiness check placeholder
// func (r *KnativeReconciler) isServiceReady(ctx context.Context, svc *servingv1.Service) (bool, error) {
// 	var current servingv1.Service
// 	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: svc.Namespace, Name: svc.Name}, &current); err != nil {
// 		return false, err
// 	}

// 	for _, cond := range current.Status.Conditions {
// 		if cond.Type == servingv1.ServiceConditionReady && cond.Status == "True" {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }

// // simple pointer helper
// func pointerBool(b bool) *bool { return &b }
