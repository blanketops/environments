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

package predicates

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
)

// mismatchedType is a stand-in for "the wrong concrete type" in every
// mismatch test below — it is never one of the switched-on kinds, so it
// always falls into the `!ok` branch of whichever case is under test.
var mismatchedType client.Object = &corev1.Pod{}

func update(old, newObj client.Object) event.UpdateEvent {
	return event.UpdateEvent{ObjectOld: old, ObjectNew: newObj}
}

// specDiffCase describes one CR kind whose predicate branch does a plain
// reflect.DeepEqual(old.Spec, new.Spec) comparison.
type specDiffCase struct {
	name   string
	newObj func(raw []byte) client.Object
}

func TestMeaningfulChangePredicate_SpecDiffKinds(t *testing.T) {
	cases := []specDiffCase{
		{"Deployment", func(raw []byte) client.Object {
			return &environmentsv1alpha1.Deployment{Spec: environmentsv1alpha1.DeploymentSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"GitHubEvent", func(raw []byte) client.Object {
			return &eventsv1alpha1.GitHubEvent{Spec: eventsv1alpha1.GitHubEventSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"GitRepository", func(raw []byte) client.Object {
			return &sourcesv1alpha1.GitRepository{Spec: sourcesv1alpha1.GitRepositorySpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"Environment", func(raw []byte) client.Object {
			return &environmentsv1alpha1.Environment{Spec: environmentsv1alpha1.EnvironmentSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"Package", func(raw []byte) client.Object {
			return &environmentsv1alpha1.Package{Spec: environmentsv1alpha1.PackageSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"ServiceUnit", func(raw []byte) client.Object {
			return &environmentsv1alpha1.ServiceUnit{Spec: environmentsv1alpha1.ServiceUnitSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"Route", func(raw []byte) client.Object {
			return &networksv1alpha1.Route{Spec: networksv1alpha1.RouteSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
		{"Domain", func(raw []byte) client.Object {
			return &networksv1alpha1.Domain{Spec: networksv1alpha1.DomainSpec{Contract: runtime.RawExtension{Raw: raw}}}
		}},
	}

	pred := MeaningfulChangePredicate()

	for _, tc := range cases {
		t.Run(tc.name+"/NoSpecChange_Suppressed", func(t *testing.T) {
			old := tc.newObj([]byte(`{"a":1}`))
			newObj := tc.newObj([]byte(`{"a":1}`))
			if pred.Update(update(old, newObj)) {
				t.Fatalf("expected identical specs to suppress reconciliation for %s", tc.name)
			}
		})

		t.Run(tc.name+"/SpecChanged_Reconciles", func(t *testing.T) {
			old := tc.newObj([]byte(`{"a":1}`))
			newObj := tc.newObj([]byte(`{"a":2}`))
			if !pred.Update(update(old, newObj)) {
				t.Fatalf("expected a spec change to trigger reconciliation for %s", tc.name)
			}
		})

		t.Run(tc.name+"/TypeMismatch_DefaultsToReconcile", func(t *testing.T) {
			old := tc.newObj([]byte(`{"a":1}`))
			if !pred.Update(update(old, mismatchedType)) {
				t.Fatalf("expected a type mismatch on ObjectNew to default to reconcile for %s", tc.name)
			}
		})
	}
}

func TestMeaningfulChangePredicate_Build_ComparesRetryAttemptAnnotation(t *testing.T) {
	pred := MeaningfulChangePredicate()

	newBuild := func(retryAttempt string, specRaw []byte) *environmentsv1alpha1.Build {
		b := &environmentsv1alpha1.Build{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
			Spec:       environmentsv1alpha1.BuildSpec{Contract: runtime.RawExtension{Raw: specRaw}},
		}
		if retryAttempt != "" {
			b.Annotations["build.blanketops.dev/retry-attempt"] = retryAttempt
		}
		return b
	}

	t.Run("SameRetryAttempt_SpecChanged_StillSuppressed", func(t *testing.T) {
		// Build's predicate branch only compares the retry-attempt
		// annotation, not Spec — a spec change alone does not reconcile.
		old := newBuild("1", []byte(`{"a":1}`))
		newObj := newBuild("1", []byte(`{"a":2}`))
		if pred.Update(update(old, newObj)) {
			t.Fatal("expected an unchanged retry-attempt annotation to suppress reconciliation regardless of spec")
		}
	})

	t.Run("RetryAttemptChanged_Reconciles", func(t *testing.T) {
		old := newBuild("1", []byte(`{"a":1}`))
		newObj := newBuild("2", []byte(`{"a":1}`))
		if !pred.Update(update(old, newObj)) {
			t.Fatal("expected a changed retry-attempt annotation to trigger reconciliation")
		}
	})

	t.Run("TypeMismatch_DefaultsToReconcile", func(t *testing.T) {
		old := newBuild("1", nil)
		if !pred.Update(update(old, mismatchedType)) {
			t.Fatal("expected a type mismatch on ObjectNew to default to reconcile")
		}
	})
}

func TestMeaningfulChangePredicate_UnknownKind_DefaultsToReconcile(t *testing.T) {
	pred := MeaningfulChangePredicate()
	old := &corev1.Pod{}
	newObj := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "changed"}}

	if !pred.Update(update(old, newObj)) {
		t.Fatal("expected an unrecognised CR kind to default to reconcile")
	}
}

func TestMeaningfulChangePredicate_CreateDeleteGenericAlwaysPassThrough(t *testing.T) {
	pred := MeaningfulChangePredicate()

	if !pred.Create(event.CreateEvent{Object: &corev1.Pod{}}) {
		t.Fatal("expected Create events to always pass through")
	}
	if !pred.Delete(event.DeleteEvent{Object: &corev1.Pod{}}) {
		t.Fatal("expected Delete events to always pass through")
	}
	if !pred.Generic(event.GenericEvent{Object: &corev1.Pod{}}) {
		t.Fatal("expected Generic events to always pass through")
	}
}
