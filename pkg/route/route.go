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

// ensureDomainMapping ensures the DomainMapping resource exists and is correctly configured.