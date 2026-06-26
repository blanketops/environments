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
	"sigs.k8s.io/controller-runtime/pkg/client"

	networksv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/networks/v1alpha1"
)

// StatusWriter persists Route conditions to the CR status.
// It does NOT derive conditions, manage contract state, or preserve domain flags.
// Callers pass the Route CR with conditions already set; this writer merges and persists.
type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("route-status-writer"),
	}
}

// Write merges the provided conditions into the Route CR's status and persists.
// The caller is responsible for fetching the Route and determining which
// conditions to apply. This method only writes.
func (w *StatusWriter) Write(ctx context.Context, route *networksv1alpha1.Route, conditions ...metav1.Condition) error {
	log := w.Log.WithValues("route", route.Name, "namespace", route.Namespace)

	for _, cond := range conditions {
		route.Status.Conditions = mergeCondition(route.Status.Conditions, cond)
		log.Info("condition merged", "type", cond.Type, "status", cond.Status, "reason", cond.Reason)
	}

	if err := w.Client.Status().Update(ctx, route); err != nil {
		log.Error(err, "failed to persist route status")
		return err
	}

	log.Info("route status persisted")
	return nil
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
