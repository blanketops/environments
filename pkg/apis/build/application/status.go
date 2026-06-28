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
	retry "k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

// StatusWriter persists Build conditions to the CR status.
// It does NOT derive conditions, manage contract state, or preserve domain flags.
// Callers pass the Build CR with conditions already set; this writer merges and persists.
type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{
		Client: c,
		Log:    log.WithName("build-status-writer"),
	}
}

// Write merges the provided conditions into the Build CR status and persists.
// Uses RetryOnConflict + re-fetch on each attempt — the buildrun-observer
// writes BuildSuccess concurrently which bumps resourceVersion between the
// reconciler's initial fetch and this write, causing conflicts. Re-fetching
// on every retry attempt guarantees we always write to the latest version.
func (w *StatusWriter) Write(ctx context.Context, build *buildv1.Build, conditions ...metav1.Condition) error {
	log := w.Log.WithValues("build", build.Name, "namespace", build.Namespace)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Re-fetch latest version — resourceVersion may have been bumped by
		// the buildrun-observer between the reconciler's initial fetch and here.
		latest := &buildv1.Build{}
		if err := w.Client.Get(ctx, client.ObjectKeyFromObject(build), latest); err != nil {
			log.Error(err, "failed to re-fetch build before status write")
			return err
		}

		for _, cond := range conditions {
			latest.Status.Conditions = mergeCondition(latest.Status.Conditions, cond)
			log.Info("condition merged", "type", cond.Type, "status", cond.Status, "reason", cond.Reason)
		}

		if err := w.Client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "failed to persist build status")
			return err
		}

		log.Info("build status persisted")
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
