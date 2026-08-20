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

package application

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
)

// StatusWriter persists ServiceUnit conditions to the CR status.
// It does NOT derive conditions or manage contract state. Callers pass the
// ServiceUnit CR with conditions already set; this writer merges and persists.
type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

// NewStatusWriter constructs a StatusWriter.
func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("serviceunit-status-writer"),
	}
}

// Write merges the provided conditions into the ServiceUnit CR status and
// persists. Uses RetryOnConflict + re-fetch on each attempt — concurrent
// writers may bump resourceVersion between the reconciler's initial fetch
// and this write. Re-fetching guarantees we always write to the latest version.
func (w *StatusWriter) Write(ctx context.Context, su *environmentv1alpha1.ServiceUnit, conditions ...metav1.Condition) error {
	log := w.Log.WithValues("serviceunit", su.Name, "namespace", su.Namespace)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &environmentv1alpha1.ServiceUnit{}
		if err := w.Client.Get(ctx, client.ObjectKeyFromObject(su), latest); err != nil {
			log.Error(err, "failed to re-fetch serviceunit before status write")
			return err
		}

		for _, cond := range conditions {
			latest.Status.Conditions = mergeCondition(latest.Status.Conditions, cond)
			log.Info("condition merged", "type", cond.Type, "status", cond.Status, "reason", cond.Reason)
		}

		if err := w.Client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "failed to persist serviceunit status")
			return err
		}

		log.Info("serviceunit status persisted")
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
