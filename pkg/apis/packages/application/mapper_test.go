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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/packages/domain"
	pkgResolution "github.com/blanketops/environments/resolution/packages/resolve"
)

func TestMapResolvedToDomain_NilStateRepository_DefaultsToPlainYAML(t *testing.T) {
	rp := &pkgResolution.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: "my-package", Namespace: "default"},
		},
		Spec: &pkgResolution.ResolvedPackageSpec{
			Enabled: true,
			Name:    "my-package",
			Version: "1.0.0",
			// StateRepository intentionally left nil — the optional-field case.
		},
	}

	got := Mapper{}.MapResolvedToDomain(rp)

	if got.Strategy != domain.StrategyPlainYAML {
		t.Fatalf("expected StrategyPlainYAML for a Package with no stateRepository, got %v", got.Strategy)
	}
	if got.StateRepo != (domain.StateRepository{}) {
		t.Fatalf("expected a zero-value StateRepo, got %+v", got.StateRepo)
	}
}
