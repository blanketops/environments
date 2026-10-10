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

package query

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	builddomain "github.com/blanketops/environments/pkg/apis/build/domain"
	serviceunitresolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

const pushedImage = "ghcr.io/example-org/app:main@sha256:1111111111111111111111111111111111111111111111111111111111111111"

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := environmentsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return scheme
}

func newBuild(t *testing.T, namespace, image string) *environmentsv1alpha1.Build {
	t.Helper()
	b := &environmentsv1alpha1.Build{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: namespace}}
	if image != "" {
		raw, err := json.Marshal(builddomain.BuildStatus{Image: image})
		if err != nil {
			t.Fatalf("marshal status: %v", err)
		}
		b.Status.Contract.Raw = raw
	}
	return b
}

func buildUnit(refNamespace string) *serviceunitresolution.ResolvedServiceUnit {
	return &serviceunitresolution.ResolvedServiceUnit{
		ServiceUnit: &environmentsv1alpha1.ServiceUnit{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}},
		Spec: &serviceunitresolution.ResolvedServiceUnitSpec{
			Type:     commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
			BuildRef: &serviceunitresolution.ResolvedBuildRef{Name: "app", Namespace: refNamespace},
		},
	}
}

func staticUnit() *serviceunitresolution.ResolvedServiceUnit {
	return &serviceunitresolution.ResolvedServiceUnit{
		ServiceUnit: &environmentsv1alpha1.ServiceUnit{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}},
		Spec: &serviceunitresolution.ResolvedServiceUnitSpec{
			Type:  commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
			Image: "ghcr.io/example-org/app:static",
		},
	}
}

func TestBuildKey(t *testing.T) {
	noRef := buildUnit("")
	noRef.Spec.BuildRef = nil

	tests := []struct {
		name   string
		su     *serviceunitresolution.ResolvedServiceUnit
		want   types.NamespacedName
		wantOK bool
	}{
		{name: "build in the serviceunit's namespace", su: buildUnit(""), want: types.NamespacedName{Namespace: "default", Name: "app"}, wantOK: true},
		{name: "build in the namespace the reference names", su: buildUnit("builds"), want: types.NamespacedName{Namespace: "builds", Name: "app"}, wantOK: true},
		{name: "static serviceunit", su: staticUnit()},
		{name: "build type without a reference", su: noRef},
		{name: "nil", su: nil},
		{name: "no object", su: &serviceunitresolution.ResolvedServiceUnit{Spec: &serviceunitresolution.ResolvedServiceUnitSpec{}}},
		{name: "no spec", su: &serviceunitresolution.ResolvedServiceUnit{ServiceUnit: &environmentsv1alpha1.ServiceUnit{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BuildKey(tt.su)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("BuildKey = %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestBuildImage(t *testing.T) {
	broken := &environmentsv1alpha1.Build{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"}}
	broken.Status.Contract.Raw = []byte(`{"Image":`)

	tests := []struct {
		name    string
		build   *environmentsv1alpha1.Build
		want    string
		wantErr string
	}{
		{name: "pushed an image", build: newBuild(t, "default", pushedImage), want: pushedImage},
		{name: "built nothing yet", build: newBuild(t, "default", "")},
		{name: "nil", build: nil},
		{name: "status that does not decode", build: broken, wantErr: "decode status of build default/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildImage(tt.build)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("BuildImage error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("BuildImage = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestInjectBuildImage(t *testing.T) {
	scheme := newScheme(t)
	ctx := context.Background()

	t.Run("takes the image its Build pushed", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newBuild(t, "default", pushedImage)).Build()
		su := buildUnit("")
		if err := InjectBuildImage(ctx, c, su); err != nil {
			t.Fatalf("InjectBuildImage: %v", err)
		}
		if su.Spec.Image != pushedImage {
			t.Errorf("image = %q, want %q", su.Spec.Image, pushedImage)
		}
	})

	t.Run("follows the namespace the reference names", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newBuild(t, "builds", pushedImage)).Build()
		su := buildUnit("builds")
		if err := InjectBuildImage(ctx, c, su); err != nil {
			t.Fatalf("InjectBuildImage: %v", err)
		}
		if su.Spec.Image != pushedImage {
			t.Errorf("image = %q, want %q", su.Spec.Image, pushedImage)
		}
	})

	// Waiting for the first build is not a failure: the image stays empty.
	t.Run("build has pushed nothing yet", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newBuild(t, "default", "")).Build()
		su := buildUnit("")
		if err := InjectBuildImage(ctx, c, su); err != nil {
			t.Fatalf("InjectBuildImage: %v", err)
		}
		if su.Spec.Image != "" {
			t.Errorf("image = %q, want it empty", su.Spec.Image)
		}
	})

	t.Run("build does not exist", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		err := InjectBuildImage(ctx, c, buildUnit(""))
		if !apierrors.IsNotFound(err) {
			t.Fatalf("InjectBuildImage = %v, want a not found error", err)
		}
	})

	t.Run("build status does not decode", func(t *testing.T) {
		broken := newBuild(t, "default", "")
		broken.Status.Contract.Raw = []byte(`{"Image":7}`)
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(broken).Build()
		if err := InjectBuildImage(ctx, c, buildUnit("")); err == nil {
			t.Fatal("InjectBuildImage = nil, want an error")
		}
	})

	// A static ServiceUnit names its own image and no Build is read.
	t.Run("static serviceunit is left as it is", func(t *testing.T) {
		var c client.Reader = fake.NewClientBuilder().WithScheme(scheme).Build()
		su := staticUnit()
		if err := InjectBuildImage(ctx, c, su); err != nil {
			t.Fatalf("InjectBuildImage: %v", err)
		}
		if su.Spec.Image != "ghcr.io/example-org/app:static" {
			t.Errorf("image = %q, want it unchanged", su.Spec.Image)
		}
	})
}
