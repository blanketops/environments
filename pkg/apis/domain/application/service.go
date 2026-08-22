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
This file owns DomainService — the application service that orchestrates the
domain reconciliation pipeline.

DomainService is the single entry point called by the domain controller after
resolution. It sequences four steps in order:

 1. Map the resolved Domain contract to a domain.Domain (Mapper).
 2. Select the correct provider (BackendSelector).
 3. Dispatch via the provider (Provider.Ensure).
 4. Write conditions back to the Domain CR status (StatusWriter).

DomainStatus is an opaque envelope — only Conditions is typed on the CR.
Scalar fields (phase, domainReady, tlsReady, certRef) are published to the
cache by the controller after Reconcile returns, via DomainCache.PublishStatus.

ErrCertProvisioning is non-terminal — returned unwrapped to the controller
as a requeue signal after conditions are written.

Condition ownership:

	DomainClaimReady   — owned by this service
	CertificateReady   — owned by this service
	DomainMappingReady — owned by the Route controller; never set here
	Ready              — aggregate; owned by this service
*/
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blanketops/environments/pkg/apis/domain/domain"
	domainResolution "github.com/blanketops/environments/resolution/domain/resolve"
)

// DomainService orchestrates the domain reconciliation pipeline.
// It is stateless beyond its collaborators and safe for concurrent use.
type DomainService struct {
	mapper  *Mapper
	status  *StatusWriter
	backend *BackendSelector
}

// NewDomainService constructs a DomainService with the required collaborators.
func NewDomainService(mapper *Mapper, status *StatusWriter, backend *BackendSelector) *DomainService {
	return &DomainService{
		mapper:  mapper,
		status:  status,
		backend: backend,
	}
}

// Reconcile executes the full domain pipeline for a resolved Domain CR.
// It maps, selects, dispatches, and writes conditions in sequence.
// Scalar status fields are published to the cache by the controller —
// not written here, because DomainStatus.Contract is opaque.
func (s *DomainService) Reconcile(ctx context.Context, resolved *domainResolution.ResolvedDomain) error {
	d := s.mapper.MapResolvedToDomain(resolved)

	provider, selErr := s.backend.ForDomain(d)
	if selErr != nil {
		conditions := s.domainConditions(d, domain.DomainResult{
			Phase:   domain.PhaseFailed,
			Message: selErr.Error(),
		}, selErr)
		return s.status.Write(ctx, resolved.Domain, conditions...)
	}

	result, ensureErr := provider.Ensure(ctx, resolved, d)

	conditions := s.domainConditions(d, result, ensureErr)
	if writeErr := s.status.Write(ctx, resolved.Domain, conditions...); writeErr != nil {
		return writeErr
	}

	// ErrCertProvisioning is a requeue signal — conditions already written.
	if errors.Is(ensureErr, domain.ErrCertProvisioning) {
		return ensureErr
	}
	return nil
}

// Teardown releases whatever Reconcile provisioned for this Domain CR. It maps
// and selects the backend exactly as Reconcile does, so it tears down the
// same provider that would have been (or was) used to materialize the
// cert/mapping chain. No status write: the CR is being deleted, so there is
// nothing left to persist status onto once this returns.
func (s *DomainService) Teardown(ctx context.Context, resolved *domainResolution.ResolvedDomain) error {
	d := s.mapper.MapResolvedToDomain(resolved)

	provider, selErr := s.backend.ForDomain(d)
	if selErr != nil {
		return selErr
	}

	return provider.Teardown(ctx, d)
}

// domainConditions derives a metav1.Condition slice from the dispatch outcome.
// This is application logic: it knows what Ready/Provisioning/Failed mean.
func (s *DomainService) domainConditions(d domain.Domain, result domain.DomainResult, runErr error) []metav1.Condition {
	now := metav1.NewTime(time.Now())

	switch {
	case errors.Is(runErr, domain.ErrCertProvisioning):
		// Non-terminal — DomainClaim already created; cert issuance in progress.
		return []metav1.Condition{
			{
				Type:               "DomainClaimReady",
				Status:             metav1.ConditionTrue,
				Reason:             "DomainClaimReady",
				Message:            "ClusterDomainClaim reconciled",
				LastTransitionTime: now,
			},
			{
				Type:               "CertificateReady",
				Status:             metav1.ConditionFalse,
				Reason:             "Provisioning",
				Message:            "certificate provisioning in progress",
				LastTransitionTime: now,
			},
			{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "CertificateProvisioning",
				Message:            "awaiting certificate issuance",
				LastTransitionTime: now,
			},
		}

	case runErr != nil:
		return []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "Failed",
				Message:            runErr.Error(),
				LastTransitionTime: now,
			},
		}

	case result.Phase == domain.PhaseReady && d.TLSStrategy == domain.TLSStrategyPlatform:
		return []metav1.Condition{
			{
				Type:               "DomainClaimReady",
				Status:             metav1.ConditionTrue,
				Reason:             "DomainClaimReady",
				Message:            "ClusterDomainClaim reconciled",
				LastTransitionTime: now,
			},
			{
				Type:               "CertificateReady",
				Status:             metav1.ConditionTrue,
				Reason:             "PlatformWildcardCoverage",
				Message:            fmt.Sprintf("platform wildcard cert covers %s", d.Host),
				LastTransitionTime: now,
			},
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "DomainReady",
				Message:            fmt.Sprintf("domain %s provisioned", d.Host),
				LastTransitionTime: now,
			},
		}

	case result.Phase == domain.PhaseReady && d.TLSStrategy == domain.TLSStrategyCustom:
		return []metav1.Condition{
			{
				Type:               "DomainClaimReady",
				Status:             metav1.ConditionTrue,
				Reason:             "DomainClaimReady",
				Message:            "ClusterDomainClaim reconciled",
				LastTransitionTime: now,
			},
			{
				Type:               "CertificateReady",
				Status:             metav1.ConditionTrue,
				Reason:             "CertificateIssued",
				Message:            fmt.Sprintf("certificate issued for %s", d.Host),
				LastTransitionTime: now,
			},
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "DomainReady",
				Message:            fmt.Sprintf("domain %s provisioned", d.Host),
				LastTransitionTime: now,
			},
		}

	default:
		return []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "Provisioning",
				Message:            result.Message,
				LastTransitionTime: now,
			},
		}
	}
}
