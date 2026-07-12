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
This file owns KnativeProvider — the Knative implementation of the routes
Provider interface.

KnativeProvider materializes a Route as a Knative DomainMapping
(serving.knative.dev/v1beta1). The DomainMapping binds the declared host to
the Knative Service identified by Route.ServiceRef, making the workload
reachable at that FQDN via Kourier.

API version:

	DomainMapping/v1alpha1 was deprecated in Knative v1.11.0. This provider
	targets the stable v1beta1 API exclusively.

TLS:

	When Route.TLSEnabled is true, the DomainMapping is created with
	spec.tls.secretName pointing to the TLS Secret that the Domain provider
	(pkg/domain/api/knative.go) provisions via cert-manager. The secret name
	follows the blanketops-tls-{sanitized-host} convention defined in common.go
	and shared between all providers. Knative serving will wait for the Secret
	to exist — no ordering dependency between Route and Domain reconciliation
	is required.

Disabled routes:

	When Route.Enabled is false the DomainMapping is deleted. The Route CR is
	retained; only the runtime resource is removed. A missing DomainMapping
	(NotFound) is not treated as an error — the disabled state is satisfied.

See also:
  - pkg/route/api/provider.go               — Provider interface
  - pkg/route/api/common.go                 — certSecretName / sanitizeHost
  - pkg/route/api/ingress.go                — Kubernetes Ingress provider
  - pkg/domain/api/knative.go               — provisions the TLS Secret this file references
  - pkg/route/application/backend_selector.go — wires KnativeProvider for RuntimeKnativeService
*/
package api

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	knservingv1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/blanketops/environments/pkg/apis/route/domain"
	routeResolution "github.com/blanketops/environments/resolution/route"
)

// KnativeProvider implements Provider for the Knative serving runtime.
// Materializes Route as a serving.knative.dev/v1beta1 DomainMapping.
type KnativeProvider struct {
	client client.Client
	log    logr.Logger
}

// NewKnativeProvider constructs a KnativeProvider with the given client and logger.
func NewKnativeProvider(c client.Client, log logr.Logger) *KnativeProvider {
	return &KnativeProvider{client: c, log: log}
}

// Ensure creates or reconciles the Knative DomainMapping for the given Route.
// When Route.Enabled is false the DomainMapping is removed instead.
func (p *KnativeProvider) Ensure(
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

	dm := &knservingv1beta1.DomainMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.Host,
			Namespace: route.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, p.client, dm, func() error {
		dm.Spec = knservingv1beta1.DomainMappingSpec{
			Ref: duckv1.KReference{
				APIVersion: "serving.knative.dev/v1",
				Kind:       "Service",
				Namespace:  route.Namespace,
				Name:       route.ServiceRef,
			},
		}
		// Wire TLS when the Route declares it. The Secret is provisioned
		// separately by the Domain provider — Knative will wait for it.
		if route.TLSEnabled {
			dm.Spec.TLS = &knservingv1beta1.SecretTLS{
				SecretName: certSecretName(route.Host),
			}
		}
		return nil
	})
	if err != nil {
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to reconcile DomainMapping %q: %v", route.Host, err),
		}, err
	}

	p.log.Info("DomainMapping reconciled",
		"host", route.Host,
		"serviceRef", route.ServiceRef,
		"tlsEnabled", route.TLSEnabled,
		"operation", op,
	)

	return domain.RouteResult{
		Phase:           domain.PhaseReady,
		ResolvedAddress: route.Host,
	}, nil
}

// ensureDisabled removes the DomainMapping when a Route is disabled.
// NotFound is not an error — the desired state is already satisfied.
func (p *KnativeProvider) ensureDisabled(ctx context.Context, route domain.Route) (domain.RouteResult, error) {
	dm := &knservingv1beta1.DomainMapping{}
	err := p.client.Get(ctx, client.ObjectKey{Name: route.Host, Namespace: route.Namespace}, dm)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return domain.RouteResult{
				Phase:   domain.PhasePending,
				Message: "route disabled; DomainMapping absent",
			}, nil
		}
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to get DomainMapping %q: %v", route.Host, err),
		}, err
	}

	if err := p.client.Delete(ctx, dm); err != nil {
		return domain.RouteResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to delete DomainMapping %q: %v", route.Host, err),
		}, err
	}

	p.log.Info("DomainMapping deleted (route disabled)", "host", route.Host)
	return domain.RouteResult{
		Phase:   domain.PhasePending,
		Message: "route disabled; DomainMapping removed",
	}, nil
}

// route/api/knative.go
func (p *KnativeProvider) Teardown(ctx context.Context, route domain.Route) error {
	dm := &knservingv1beta1.DomainMapping{}
	err := p.client.Get(ctx, client.ObjectKey{Name: route.Host, Namespace: route.Namespace}, dm)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("get domainmapping %q: %w", route.Host, err)
	}
	if err != nil {
		return nil
	}
	if err := p.client.Delete(ctx, dm); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete domainmapping %q: %w", route.Host, err)
	}
	p.log.Info("DomainMapping deleted (teardown)", "host", route.Host)
	return nil
}
