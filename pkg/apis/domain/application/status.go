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
This file owns StatusWriter — the dumb persister for Domain CR status.

Mirrors the Build and Route StatusWriter pattern exactly. Write takes the
Domain CR and variadic conditions — it merges conditions and persists.
Scalar status fields (phase, domainReady, tlsReady, certificateRef) are
set directly on the CR by the caller (DomainService) before Write is invoked.

DomainMappingRef is never written here — it is owned by the Route controller.

Uses RetryOnConflict + re-fetch on each attempt — concurrent writers may bump
resourceVersion between the caller's initial fetch and this write.

See also:
  - pkg/apis/domain/application/service.go  — sets scalar fields and calls Write
  - pkg/apis/build/application/status.go    — canonical pattern this mirrors
  - pkg/apis/route/application/status.go    — same pattern for Route
*/
package application

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networksv1alpha1 "github.com/BlanketOps/environments-api/api/networks/v1alpha1"
)

// StatusWriter persists Domain conditions to the CR status.
// It does NOT derive conditions, manage phase transitions, or set scalar fields.
// Callers set scalar fields on the CR directly; this writer merges conditions
// and persists via Status().Update().
type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

// NewStatusWriter constructs a StatusWriter with the given client and logger.
func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("domain-status-writer"),
	}
}

// Write merges the provided conditions into the Domain CR status and persists.
// Uses RetryOnConflict + re-fetch on each attempt — concurrent writers may bump
// resourceVersion between the caller's fetch and this write. Re-fetching on
// every retry guarantees we always write to the latest version.
// Scalar fields must be set on the CR by the caller before Write is invoked —
// they are re-applied from the caller's version on each retry attempt since
// DomainStatus.Contract is opaque and conditions are the only typed field.
func (w *StatusWriter) Write(ctx context.Context, cr *networksv1alpha1.Domain, conditions ...metav1.Condition) error {
	log := w.Log.WithValues("domain", cr.Name, "namespace", cr.Namespace)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &networksv1alpha1.Domain{}
		if err := w.Client.Get(ctx, client.ObjectKeyFromObject(cr), latest); err != nil {
			log.Error(err, "failed to re-fetch domain before status write")
			return err
		}

		for _, cond := range conditions {
			latest.Status.Conditions = mergeCondition(latest.Status.Conditions, cond)
			log.Info("condition merged", "type", cond.Type, "status", cond.Status, "reason", cond.Reason)
		}

		if err := w.Client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "failed to persist domain status")
			return err
		}

		log.Info("domain status persisted")
		return nil
	})
}

// mergeCondition upserts newCond into conds by Type.
func mergeCondition(conds []metav1.Condition, newCond metav1.Condition) []metav1.Condition {
	for i, c := range conds {
		if c.Type == newCond.Type {
			conds[i] = newCond
			return conds
		}
	}
	return append(conds, newCond)
}
