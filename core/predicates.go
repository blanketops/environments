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

package core

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	environmentsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
)

// MeaningfulChangePredicate returns predicate funcs that ONLY
// Other changes (status updates, metadata changes) are ignored.
func MeaningfulChangePredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			switch old := e.ObjectOld.(type) {

			case *environmentsv1alpha1.Build:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.Build)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *environmentsv1alpha1.BuildTrigger:
				newObj, ok := e.ObjectNew.(*environmentsv1alpha1.BuildTrigger)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

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

			default:
				// Unknown type → reconcile (safe default)
				return true
			}
		},

		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}
}
