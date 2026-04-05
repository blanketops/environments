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

package route

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler is responsible for reconciling Knative Routes.
type RouteReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewRouteReconciler creates a new reconciler.
func NewRouteReconciler(c client.Client, scheme *runtime.Scheme, log logr.Logger) *RouteReconciler {
	return &RouteReconciler{Client: c, Scheme: scheme, Log: log}
}

// ensureKnativeConfigDomain patches the `config-domain` ConfigMap to include the desired domain.
// func (r *RouteReconciler) ensureKnativeConfigDomain(ctx context.Context, env *environmentv1.Environment) error {
// 	const configMapName = "config-domain"
// 	const configMapNamespace = "knative-serving"

// 	// Get the Knative config-domain ConfigMap.
// 	configMap := &corev1.ConfigMap{}
// 	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: configMapNamespace, Name: configMapName}, configMap); err != nil {
// 		if apierrors.IsNotFound(err) {
// 			r.Log.Info("Knative config-domain ConfigMap not found, skipping patch.")
// 			return nil
// 		}
// 		return fmt.Errorf("getting Knative config-domain ConfigMap: %w", err)
// 	}

// 	// Check if the domain is already configured.
// 	domainKey := env.Spec.Route.Host
// 	if strings.Contains(domainKey, ".") {
// 		domainKey = strings.Join(strings.Split(domainKey, ".")[1:], ".")
// 	} else {
// 		domainKey = "" // Don't patch if it's a simple host, Knative can handle this
// 	}

// 	if domainKey == "" {
// 		return nil
// 	}

// 	if _, ok := configMap.Data[domainKey]; ok {
// 		r.Log.Info("Knative domain already configured", "domain", domainKey)
// 		return nil
// 	}

// 	// Prepare the patch to update the ConfigMap.
// 	patch := map[string]interface{}{
// 		"data": map[string]string{
// 			env.Spec.Route.Host: "",
// 		},
// 	}
// 	patchBytes, err := json.Marshal(patch)
// 	if err != nil {
// 		return fmt.Errorf("marshalling patch for ConfigMap: %w", err)
// 	}

// 	r.Log.Info("Patching Knative config-domain ConfigMap", "domain", env.Spec.Route.Host)
// 	if err := r.Client.Patch(ctx, configMap, client.RawPatch(types.StrategicMergePatchType, patchBytes)); err != nil {
// 		return fmt.Errorf("patching Knative config-domain ConfigMap: %w", err)
// 	}

// 	return nil
// }

//func (r *RouteReconciler) ensureDomainMapping(ctx context.Context, env *environmentv1.Environment, knativeService string) error {
// Create the DomainMapping object.
// domainMapping := &unstructured.Unstructured{
// 	Object: map[string]interface{}{
// 		"apiVersion": "serving.knative.dev/v1beta1",
// 		"kind":       "DomainMapping",
// 		"metadata": map[string]interface{}{
// 			"name":      env.Spec.Route.Spec.Host,
// 			"namespace": env.Namespace,
// 		},
// 		"spec": map[string]interface{}{
// 			"ref": map[string]interface{}{
// 				"name":       knativeService,
// 				"kind":       "Service",
// 				"apiVersion": "serving.knative.dev/v1",
// 			},
// 		},
// 	},
// }

// // Set the owner reference.
// if err := controllerutil.SetControllerReference(env, domainMapping, r.Scheme); err != nil {
// 	return fmt.Errorf("setting owner reference on DomainMapping: %w", err)
// }

// // Use CreateOrUpdate to ensure the resource exists and is in the desired state.
// found := &unstructured.Unstructured{}
// found.SetGroupVersionKind(domainMapping.GroupVersionKind())

// op, err := controllerutil.CreateOrUpdate(ctx, r.Client, found, func() error {
// 	found.Object = domainMapping.Object
// 	return nil
// })

// if err != nil {
// 	return fmt.Errorf("creating or updating DomainMapping: %w", err)
// }

// if op != controllerutil.OperationResultNone {
// 	r.Log.Info("DomainMapping successfully created or updated", "operation", op)
// }

//return nil
//}
