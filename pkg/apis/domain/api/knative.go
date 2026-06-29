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
This file owns KnativeProvider — the implementation of the domain Provider
interface that handles both TLS strategies via Knative networking and
cert-manager.

Strategy dispatch:

	platform → ensurePlatform: ClusterDomainClaim only.
	           The platform DNS01 wildcard cert (*.<zone>) already covers this
	           host via a ClusterIssuer. No Certificate resource is created.

	custom   → ensureCustom: Issuer + ClusterDomainClaim + Certificate.
	           A namespaced HTTP01 Issuer is created per tenant namespace.
	           An ACME Certificate is issued via the nginx solver. The controller
	           requeues (ErrCertProvisioning) until the Certificate reaches Ready.

ClusterDomainClaim:

	Knative's networking.internal.knative.dev/v1alpha1 ClusterDomainClaim is
	cluster-scoped (no namespace). It delegates domain ownership to a specific
	namespace via spec.namespace. Created for both TLS strategies.

TLS Secret naming:

	The cert-manager Certificate writes its secret as blanketops-tls-{host}
	(sanitized). The Route provider (pkg/routes/api/knative.go) references this
	same name in DomainMapping.Spec.TLS.SecretName — a shared naming convention
	avoids any cross-package coupling.

See also:
  - pkg/domain/api/provider.go         — Provider interface
  - pkg/domain/domain/model.go         — TLSStrategy constants
  - pkg/routes/api/knative.go          — references certSecretName(host) for TLS wiring
  - pkg/domain/application/backend_selector.go — wires this provider
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knnetworkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/domain/domain"
	domainResolution "github.com/ntlaletsi70/blanketops-environments/resolution/domain"
)

// ACMEConfig carries the HTTP01 ACME issuer configuration injected at
// construction time. Keeps provider logic free of hardcoded platform values.
type ACMEConfig struct {
	// Server is the ACME directory URL (e.g. Let's Encrypt production endpoint).
	Server string
	// Email is the ACME account email for cert expiry notifications.
	Email string
	// PrivateKeySecretName is the name of the Secret storing the ACME account
	// private key. Created in the target namespace by cert-manager.
	PrivateKeySecretName string
}

// KnativeProvider implements Provider for the Knative + cert-manager stack.
type KnativeProvider struct {
	client client.Client
	log    logr.Logger
	acme   ACMEConfig
}

// NewKnativeProvider constructs a KnativeProvider with the given dependencies.
func NewKnativeProvider(c client.Client, log logr.Logger, acme ACMEConfig) *KnativeProvider {
	return &KnativeProvider{client: c, log: log, acme: acme}
}

// Ensure provisions the cert/mapping chain for the given Domain.
// Dispatches to ensurePlatform or ensureCustom based on TLSStrategy.
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

// ── platform strategy ─────────────────────────────────────────────────────────

// ensurePlatform handles the platform wildcard cert path.
// Emits ClusterDomainClaim only — the DNS01 ClusterIssuer wildcard already
// covers this host. No Certificate resource is needed.
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
		CertIssued: true, // platform wildcard already covers this host
	}, nil
}

// ── custom strategy ───────────────────────────────────────────────────────────

// ensureCustom handles the client-owned zone path.
// Emits Issuer + ClusterDomainClaim + Certificate. The controller requeues
// (ErrCertProvisioning) until the Certificate reaches Ready.
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
		// ACME challenge or cert issuance still in progress — requeue.
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

// ── resource reconciliation ───────────────────────────────────────────────────

// ensureClusterDomainClaim creates or reconciles the Knative ClusterDomainClaim
// for this host. ClusterDomainClaim is cluster-scoped — spec.namespace delegates
// ownership of the domain to the tenant namespace.
func (p *KnativeProvider) ensureClusterDomainClaim(ctx context.Context, d domain.Domain) error {
	cdc := &knnetworkingv1alpha1.ClusterDomainClaim{
		ObjectMeta: metav1.ObjectMeta{
			// ClusterDomainClaim is cluster-scoped — no Namespace in ObjectMeta.
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

// ensureIssuer creates or reconciles a namespaced cert-manager HTTP01 Issuer
// in the tenant namespace. One Issuer per namespace covers all custom domains.
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

// ensureCertificate creates or reconciles the cert-manager Certificate for
// this host. Returns (true, nil) when the cert is Ready, (false, nil) when
// provisioning is still in progress, or (false, err) on failure.
func (p *KnativeProvider) ensureCertificate(
	ctx context.Context,
	resolved *domainResolution.ResolvedDomain,
	d domain.Domain,
) (bool, error) {
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
			IssuerRef: cmmeta.ObjectReference{ //nolint:staticcheck
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

	// Check cert readiness from the status conditions.
	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady {
			return cond.Status == cmmeta.ConditionTrue, nil
		}
	}
	// No Ready condition yet — still provisioning.
	return false, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// issuerName returns the shared HTTP01 Issuer name for a namespace.
func issuerName(namespace string) string {
	return fmt.Sprintf("blanketops-http01-%s", namespace)
}

// certName returns the cert-manager Certificate CR name for a host.
func certName(host string) string {
	return fmt.Sprintf("blanketops-cert-%s", sanitizeHost(host))
}

// certSecretName returns the TLS Secret name cert-manager writes into.
// Shared convention with pkg/routes/api/knative.go certSecretName().
func certSecretName(host string) string {
	return fmt.Sprintf("blanketops-tls-%s", sanitizeHost(host))
}

// sanitizeHost replaces dots with dashes to produce a valid Kubernetes name.
func sanitizeHost(host string) string {
	return strings.ReplaceAll(host, ".", "-")
}

func stringPtr(s string) *string { return &s }
