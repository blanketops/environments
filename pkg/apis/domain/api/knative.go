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
This file owns KnativeProvider — the Provider implementation for the domain
API that provisions Knative networking and cert-manager resources across TLS
strategies.

Strategy dispatch:

  - platform → ensurePlatform: ClusterDomainClaim only. The platform
    wildcard certificate already covers the host; no Issuer or
    Certificate resource is created.
  - custom   → ensureCustom: Issuer + ClusterDomainClaim + Certificate.
    Provisions a per-namespace HTTP01 Issuer and per-host
    Certificate, requeuing (ErrCertProvisioning) until Ready.

Resource & naming conventions:

  - ClusterDomainClaim is cluster-scoped and named after d.Host, with
    spec.namespace delegating ownership to the target namespace.
  - Certificate and TLS Secret names derive from sanitizeHost(d.Host)
    (blanketops-cert-{host} and blanketops-tls-{host}) so route wiring
    can reference the Secret without cross-package coupling.

See also:
  - pkg/domain/api/provider.go               — Provider interface
  - pkg/domain/domain/model.go               — TLSStrategy constants & errors
  - pkg/routes/api/knative.go                — consumes TLS Secret name
  - pkg/domain/application/backend_selector.go — selects backend provider
*/
package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	cmacme "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knnetworkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/blanketops/environments/pkg/apis/domain/domain"
	domainResolution "github.com/blanketops/environments/resolution/domain/resolve"
)

// ACMEConfig holds HTTP01 ACME issuer settings injected at construction time
// so provider logic stays free of hardcoded environment values[cite: 2].
type ACMEConfig struct {
	Server               string
	Email                string
	PrivateKeySecretName string
}

// KnativeProvider reconciles and tears down Knative ClusterDomainClaim and
// cert-manager Issuer/Certificate resources for a Domain[cite: 2].
type KnativeProvider struct {
	client client.Client
	log    logr.Logger
	acme   ACMEConfig
}

// NewKnativeProvider constructs a KnativeProvider[cite: 2].
func NewKnativeProvider(c client.Client, log logr.Logger, acme ACMEConfig) *KnativeProvider {
	return &KnativeProvider{client: c, log: log, acme: acme}
}

// Ensure provisions or reconciles the domain's networking and certificate
// resources according to d.TLSStrategy[cite: 2].
func (p *KnativeProvider) Ensure(
	ctx context.Context,
	resolved *domainResolution.ResolvedDomain,
	d domain.Domain,
) (domain.DomainResult, error) {
	switch d.TLSStrategy {
	case domain.TLSStrategyPlatform:
		return p.ensurePlatform(ctx, d)
	case domain.TLSStrategyCustom:
		return p.ensureCustom(ctx, resolved, d)
	default:
		return domain.DomainResult{
			Phase:   domain.PhaseFailed,
			Message: domain.ErrStrategyUnknown.Error(),
		}, domain.ErrStrategyUnknown
	}
}

func (p *KnativeProvider) ensurePlatform(ctx context.Context, d domain.Domain) (domain.DomainResult, error) {
	if err := p.ensureClusterDomainClaim(ctx, d); err != nil {
		return domain.DomainResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to reconcile ClusterDomainClaim: %v", err),
		}, err
	}

	p.log.Info("platform domain provisioned", "host", d.Host, "namespace", d.Namespace)
	return domain.DomainResult{
		Phase:      domain.PhaseReady,
		CertIssued: true,
	}, nil
}

func (p *KnativeProvider) ensureCustom(
	ctx context.Context,
	resolved *domainResolution.ResolvedDomain,
	d domain.Domain,
) (domain.DomainResult, error) {
	if err := p.ensureIssuer(ctx, d); err != nil {
		return domain.DomainResult{
			Phase:   domain.PhaseProvisioning,
			Message: fmt.Sprintf("failed to reconcile Issuer: %v", err),
		}, err
	}

	if err := p.ensureClusterDomainClaim(ctx, d); err != nil {
		return domain.DomainResult{
			Phase:   domain.PhaseFailed,
			Message: fmt.Sprintf("failed to reconcile ClusterDomainClaim: %v", err),
		}, err
	}

	certReady, err := p.ensureCertificate(ctx, resolved, d)
	if err != nil {
		return domain.DomainResult{
			Phase:   domain.PhaseProvisioning,
			Message: fmt.Sprintf("failed to reconcile Certificate: %v", err),
		}, err
	}

	if !certReady {
		// ACME issuance is asynchronous; requeue until Ready is True[cite: 2].
		return domain.DomainResult{
			Phase:   domain.PhaseProvisioning,
			Message: "certificate provisioning in progress",
		}, domain.ErrCertProvisioning
	}

	p.log.Info("custom domain provisioned", "host", d.Host)
	return domain.DomainResult{
		Phase:      domain.PhaseReady,
		CertIssued: true,
	}, nil
}

func (p *KnativeProvider) ensureClusterDomainClaim(ctx context.Context, d domain.Domain) error {
	cdc := &knnetworkingv1alpha1.ClusterDomainClaim{
		ObjectMeta: metav1.ObjectMeta{
			// Cluster-scoped resource; ownership is delegated via Spec.Namespace[cite: 2].
			Name: d.Host,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, p.client, cdc, func() error {
		cdc.Spec = knnetworkingv1alpha1.ClusterDomainClaimSpec{
			Namespace: d.Namespace,
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.log.V(1).Info("ClusterDomainClaim reconciled", "host", d.Host, "namespace", d.Namespace, "operation", op)
	return nil
}

func (p *KnativeProvider) ensureIssuer(ctx context.Context, d domain.Domain) error {
	issuer := &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      issuerName(d.Namespace),
			Namespace: d.Namespace,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, p.client, issuer, func() error {
		issuer.Spec = certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				ACME: &cmacme.ACMEIssuer{
					Server: p.acme.Server,
					Email:  p.acme.Email,
					PrivateKey: cmmeta.SecretKeySelector{
						LocalObjectReference: cmmeta.LocalObjectReference{
							Name: p.acme.PrivateKeySecretName,
						},
					},
					Solvers: []cmacme.ACMEChallengeSolver{
						{
							HTTP01: &cmacme.ACMEChallengeSolverHTTP01{
								Ingress: &cmacme.ACMEChallengeSolverHTTP01Ingress{
									Class: stringPtr("nginx"),
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.log.V(1).Info("Issuer reconciled", "name", issuerName(d.Namespace), "operation", op)
	return nil
}

func (p *KnativeProvider) ensureCertificate(
	ctx context.Context,
	resolved *domainResolution.ResolvedDomain,
	d domain.Domain,
) (bool, error) {
	_ = resolved
	cert := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName(d.Host),
			Namespace: d.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, p.client, cert, func() error {
		spec := certmanagerv1.CertificateSpec{
			DNSNames:   []string{d.Host},
			SecretName: certSecretName(d.Host),
			IssuerRef: cmmeta.IssuerReference{
				Name: issuerName(d.Namespace),
				Kind: "Issuer",
			},
		}
		if d.RenewBefore != "" {
			if dur, parseErr := time.ParseDuration(d.RenewBefore); parseErr == nil {
				spec.RenewBefore = &metav1.Duration{Duration: dur}
			}
		}
		cert.Spec = spec
		return nil
	})
	if err != nil {
		return false, err
	}
	p.log.V(1).Info("Certificate reconciled", "host", d.Host, "operation", op)

	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady {
			return cond.Status == cmmeta.ConditionTrue, nil
		}
	}
	return false, nil
}

// Teardown deletes the per-host ClusterDomainClaim and, for custom-strategy
// domains, the per-host Certificate. The per-namespace Issuer is shared
// across custom domains in that namespace and is intentionally left intact.
// Idempotent — a missing ClusterDomainClaim or Certificate is not an error[cite: 2].
func (p *KnativeProvider) Teardown(ctx context.Context, d domain.Domain) error {
	if d.Host == "" {
		return fmt.Errorf("domain host is required for teardown")
	}

	cdc := &knnetworkingv1alpha1.ClusterDomainClaim{
		ObjectMeta: metav1.ObjectMeta{Name: d.Host},
	}
	if err := p.client.Delete(ctx, cdc); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete clusterdomainclaim %q: %w", d.Host, err)
	}

	if d.TLSStrategy == domain.TLSStrategyCustom && d.Namespace != "" {
		cert := &certmanagerv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certName(d.Host),
				Namespace: d.Namespace,
			},
		}
		if err := p.client.Delete(ctx, cert); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete certificate %q: %w", d.Host, err)
		}
	}

	p.log.Info("domain teardown complete", "host", d.Host, "strategy", d.TLSStrategy)
	return nil
}

func issuerName(namespace string) string {
	return fmt.Sprintf("blanketops-http01-%s", namespace)
}

func certName(host string) string {
	return fmt.Sprintf("blanketops-cert-%s", sanitizeHost(host))
}

func certSecretName(host string) string {
	return fmt.Sprintf("blanketops-tls-%s", sanitizeHost(host))
}

func sanitizeHost(host string) string {
	return strings.ReplaceAll(host, ".", "-")
}

func stringPtr(s string) *string { return &s }
