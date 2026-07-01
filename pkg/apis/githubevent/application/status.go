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
This file owns StatusWriter for the GitHubEvent domain — the component
responsible for persisting provider results back to the GitHubEvent CR.

Two writes happen in a single Status().Update call:

 1. Contract status — JSON-marshalled domain status written to Status.Contract.
    Authoritative and machine-readable.

 2. Conditions — a single metav1.Condition upserted for kubectl describe
    alignment and human observability.

StatusWriter is always called regardless of provider error so the CR always
reflects the latest outcome.
*/
package application

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventsv1alpha1 "github.com/BlanketOps/environments-api/api/events/v1alpha1"
)

type StatusWriter struct {
	Client client.Client
	Log    logr.Logger
}

func NewStatusWriter(c client.Client, log logr.Logger) *StatusWriter {
	return &StatusWriter{Client: c, Log: log.WithName("githubevent-status-writer")}
}

// Write merges the provided conditions and persists. Nothing else.
func (w *StatusWriter) Write(ctx context.Context, gh *eventsv1alpha1.GitHubEvent, conditions ...metav1.Condition) error {
	log := w.Log.WithValues("githubevent", gh.Name, "namespace", gh.Namespace)

	for _, cond := range conditions {
		gh.Status.Conditions = mergeCondition(gh.Status.Conditions, cond)
		log.Info("condition merged", "type", cond.Type, "status", cond.Status, "reason", cond.Reason)
	}

	if err := w.Client.Status().Update(ctx, gh); err != nil {
		log.Error(err, "failed to persist githubevent status")
		return err
	}
	log.Info("githubevent status persisted")
	return nil
}

func mergeCondition(conds []metav1.Condition, newCond metav1.Condition) []metav1.Condition {
	for i, c := range conds {
		if c.Type == newCond.Type {
			conds[i] = newCond
			return conds
		}
	}
	return append(conds, newCond)
}
