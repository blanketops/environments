package core

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	buildtriggerv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	buildv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	deploymentv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	packagev1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	serviceunitv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
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

			case *buildtriggerv1alpha1.BuildTrigger:
				newObj, ok := e.ObjectNew.(*buildtriggerv1alpha1.BuildTrigger)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *deploymentv1alpha1.Deployment:
				newObj, ok := e.ObjectNew.(*deploymentv1alpha1.Deployment)
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

			case *packagev1alpha1.Package:
				newObj, ok := e.ObjectNew.(*packagev1alpha1.Package)
				if !ok {
					return true
				}
				return !reflect.DeepEqual(old.Spec, newObj.Spec)

			case *serviceunitv1alpha1.ServiceUnit:
				newObj, ok := e.ObjectNew.(*serviceunitv1alpha1.ServiceUnit)
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
