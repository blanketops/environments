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
Package api — ingress.go

IngressProvider is the Kubernetes Ingress implementation of the routes
Provider interface.

IngressProvider materializes a Route as a networking.k8s.io/v1 Ingress
resource, routing traffic from the declared host and path to the K8s Service
identified by Route.ServiceRef via the nginx ingress controller.

Ingress class:

	All BlanketOps Ingress resources use the "nginx" IngressClassName.
	Kourier handles all Knative traffic — nginx is strictly for
	RuntimeKubernetesContainer routes.

Service port convention:

	BlanketOps K8s Services expose port 80 regardless of container port.
	The Ingress backend always targets port 80. If a ServiceUnit declares a
	non-standard service port in future, domain.Route.ServicePort carries it.

TLS:

	When Route.TLSEnabled is true, Ingress.Spec.TLS is populated with the
	shared blanketops-tls-{sanitized-host} secret name (see common.go).
	The Domain provider provisions the Secret via cert-manager — no ordering
	dependency between Route and Domain reconciliation is required; nginx
	will serve without TLS until the Secret appears, then upgrade.

Disabled routes:

	When Route.Enabled is false the Ingress is deleted. The Route CR is
	retained; only the runtime resource is removed. A missing Ingress
	(NotFound) is not treated as an error — the disabled state is satisfied.

See also:
  - pkg/route/api/provider.go              — Provider interface
  - pkg/route/api/knative.go               — Knative DomainMapping provider
  - pkg/route/api/common.go                — certSecretName / sanitizeHost
  - pkg/domain/api/knative.go              — provisions the TLS Secret
  - pkg/route/application/backend_selector.go — wires IngressProvider for RuntimeKubernetesContainer
*/
package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/blanketops/environments/pkg/apis/route/domain"
	routeResolution "github.com/blanketops/environments/resolution/route/resolve"
)

const (
	// ingressClassName is the IngressClass used for all RuntimeKubernetesContainer
	// routes. Kourier owns Knative traffic — nginx owns everything else.
	ingressClassName = "nginx"

	// ingressServicePort is the port on the K8s Service backing the ServiceUnit.
	// Convention: BlanketOps Services expose port 80 regardless of container port.
	ingressServicePort int32 = 80
)

// IngressProvider implements Provider for the Kubernetes Ingress runtime.
// Materializes Route as a networking.k8s.io/v1 Ingress resource.
type IngressProvider struct {
	client client.Client
	log    logr.Logger
}

// NewIngressProvider constructs an IngressProvider with the given client and logger.
func NewIngressProvider(c client.Client, log logr.Logger) *IngressProvider {
	return &IngressProvider{client: c, log: log}
}

// Ensure creates or reconciles the Kubernetes Ingress for the given Route.
// When Route.Enabled is false the Ingress is removed instead.
func (p *IngressProvider) Ensure(
	ctx context.Context,
	resolved *routeResolution.ResolvedRoute,
	route domain.Route,
) (domain.RouteResult, error) {
	if !route.Enabled {
		return p.ensureDisabled(ctx, route)
	}

	if route.ServiceRef == "" {
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: domain.ErrServiceRefEmpty.Error(),
		}, domain.ErrServiceRefEmpty
	}

	pathType := networkingv1.PathTypePrefix
	ingressClass := ingressClassName
	servicePort := ingressServicePort

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressResourceName(route.Host),
			Namespace: route.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, p.client, ing, func() error {
		ing.Spec.IngressClassName = &ingressClass
		ing.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: route.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     route.Path,
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: route.ServiceRef,
										Port: networkingv1.ServiceBackendPort{
											Number: servicePort,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Wire TLS when the Route declares it. The Secret is provisioned
		// separately by the Domain provider — nginx will serve without TLS
		// until the Secret appears, then upgrade automatically.
		if route.TLSEnabled {
			ing.Spec.TLS = []networkingv1.IngressTLS{
				{
					Hosts:      []string{route.Host},
					SecretName: certSecretName(route.Host),
				},
			}
		}

		return nil
	})

	if err != nil {
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to reconcile Ingress %q: %v", route.Host, err),
		}, err
	}

	p.log.Info("Ingress reconciled",
		"host", route.Host,
		"serviceRef", route.ServiceRef,
		"path", route.Path,
		"tlsEnabled", route.TLSEnabled,
		"operation", op,
	)

	return domain.RouteResult{
		Phase:           domain.PhaseReady,
		ResolvedAddress: route.Host,
	}, nil
}

// ensureDisabled removes the Ingress when a Route is disabled.
// NotFound is not an error — the desired state is already satisfied.
func (p *IngressProvider) ensureDisabled(ctx context.Context, route domain.Route) (domain.RouteResult, error) {
	ing := &networkingv1.Ingress{}
	err := p.client.Get(ctx, client.ObjectKey{
		Name:      ingressResourceName(route.Host),
		Namespace: route.Namespace,
	}, ing)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return domain.RouteResult{
				Phase:   domain.PhasePending,
				Message: "route disabled; Ingress absent",
			}, nil
		}
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to get Ingress %q: %v", route.Host, err),
		}, err
	}

	if err := p.client.Delete(ctx, ing); err != nil {
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to delete Ingress %q: %v", route.Host, err),
		}, err
	}

	p.log.Info("Ingress deleted (route disabled)", "host", route.Host)

	return domain.RouteResult{
		Phase:   domain.PhasePending,
		Message: "route disabled; Ingress removed",
	}, nil
}

// ingressResourceName returns the Kubernetes Ingress resource name for a host.
// Prefixed to distinguish from DomainMapping resources sharing the namespace.
// e.g. api.dev.blanketops.online → blanketops-route-api-dev-blanketops-online
func ingressResourceName(host string) string {
	return fmt.Sprintf("blanketops--environment-route-%s", sanitizeHost(host))
}

// Teardown deletes the Ingress for the route's host, if one exists.
func (p *IngressProvider) Teardown(ctx context.Context, route domain.Route) error {
	ing := &networkingv1.Ingress{}
	err := p.client.Get(ctx, client.ObjectKey{
		Name:      ingressResourceName(route.Host),
		Namespace: route.Namespace,
	}, ing)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("get ingress %q: %w", route.Host, err)
	}
	if err != nil {
		return nil // already absent
	}
	if err := p.client.Delete(ctx, ing); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete ingress %q: %w", route.Host, err)
	}
	p.log.Info("Ingress deleted (teardown)", "host", route.Host)
	return nil
}
