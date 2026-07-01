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
Package core defines the foundational contracts and primitives shared across
all BlanketOps Environments domains.

This file owns controller-runtime predicate filtering. Predicates are the
first line of defence against unnecessary reconciliation — they run in the
informer goroutine before a reconcile request ever enters the work queue.

The key filtering concern for BlanketOps is spec-only reconciliation: status
updates and metadata changes (labels, annotations, finalizers) must not
trigger domain execution. Without this filter, a status patch written by a
domain at the end of reconciliation would re-enqueue the same object,
creating a tight reconciliation loop.

MeaningfulChangePredicate enforces this by type-switching on the concrete CR
kind and comparing specs directly. All other event types (Create, Delete,
Generic) pass through unconditionally — only Update events are filtered.
*/
package core

import (
	"reflect"

	environmentsv1alpha1 "github.com/BlanketOps/environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/BlanketOps/environments-api/api/events/v1alpha1"
	networksv1alpha1 "github.com/BlanketOps/environments-api/api/networks/v1alpha1"
	sourcesv1alpha1 "github.com/BlanketOps/environments-api/api/sources/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// MeaningfulChangePredicate returns a predicate.Funcs that suppresses Update
// events where only status or metadata changed. Spec changes always pass
// through; all other event types (Create, Delete, Generic) are unconditional.
//
// This predicate is registered at the manager level and applied to all
// BlanketOps CR controllers. It is the primary guard against reconciliation
// loops caused by status writes re-enqueuing the same object.
//
// The type switch covers all ten CR kinds managed by the platform. Unknown
// types fall through to true (reconcile) as a safe default — an unknown kind
// reaching this predicate indicates a registration gap, not a reason to drop
// the event.
func MeaningfulChangePredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			switch old := e.ObjectOld.(type) {

			case *environmentsv1alpha1.Build:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.Build)
				if !ok {
					return true
				}
				return old.Annotations["build.blanketops.dev/retry-attempt"] != newObj.Annotations["build.blanketops.dev/retry-attempt"]

			case *environmentsv1alpha1.Deployment:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.Deployment)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *eventsv1alpha1.GitHubEvent:
				newObj, ok := e.ObjectNew.(*eventsv1alpha1.GitHubEvent)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *sourcesv1alpha1.GitRepository:
				newObj, ok := e.ObjectNew.(*sourcesv1alpha1.GitRepository)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *environmentsv1alpha1.Environment:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.Environment)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *environmentsv1alpha1.Package:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.Package)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *environmentsv1alpha1.ServiceUnit:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.ServiceUnit)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)
			case *networksv1alpha1.Route:
				newObj, ok := e.ObjectNew.(*networksv1alpha1.Route)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)
			case *networksv1alpha1.Domain:
				newObj, ok := e.ObjectNew.(*networksv1alpha1.Domain)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			default:
				// Unknown kind — reconcile as safe default. An unknown type
				// reaching this predicate indicates a missing case, not a
				// reason to silently drop the event.
				return true
			}
		},

		// Create, Delete, and Generic events always pass through — there is
		// no spec-diff concern for these event types.
		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}
}
