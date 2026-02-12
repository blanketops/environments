package core

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	buildv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"
)

// MeaningfulChangePredicate returns predicate funcs that ONLY
// Other changes (status updates, metadata changes) are ignored.
func MeaningfulChangePredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			switch old := e.ObjectOld.(type) {

			case *buildv1alpha1.Build:
				newObj, ok := e.ObjectNew.(*buildv1alpha1.Build)
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
