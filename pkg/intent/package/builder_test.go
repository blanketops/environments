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

package intent

import (
	"testing"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/resolution/packages/resolve"
)

func newResolvedPackage(labels map[string]string) *resolve.ResolvedPackage {
	return &resolve.ResolvedPackage{
		Package: &environmentv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{Name: "app-package", Namespace: "default", UID: types.UID("uid-package"), Labels: labels},
		},
		Spec: &resolve.ResolvedPackageSpec{
			Enabled:           true,
			Name:              "app",
			Version:           "v1.2.3",
			PackageRepository: resolve.ResolvedPackageRepository{URL: "git@github.com:example-org/packages.git", CredentialsSecret: "packages-creds"},
			StateRepository: &resolve.ResolvedStateRepository{
				URL: "git@github.com:example-org/state.git", Ref: "master", CloneSecret: "state-creds", Strategy: "kustomization", Path: "./clusters/dev",
			},
		},
	}
}

func TestBuildPackageIntent_CarriesIdentityOwnerAndLabels(t *testing.T) {
	in, err := BuildPackageIntent(newResolvedPackage(map[string]string{
		"environments.blanketops.dev/name": "app",
		"environments.blanketops.dev/type": "dev",
		"app.kubernetes.io/managed-by":     "kustomize",
	}))
	if err != nil {
		t.Fatalf("BuildPackageIntent: %v", err)
	}
	if in.ID.Name != "app-package" || in.ID.Namespace != "default" || in.OwnerUID != "uid-package" {
		t.Errorf("identity = %+v owner = %q", in.ID, in.OwnerUID)
	}
	if len(in.Labels) != 2 || in.Labels["environments.blanketops.dev/name"] != "app" || in.Labels["environments.blanketops.dev/type"] != "dev" {
		t.Errorf("labels = %v, want only the environments.blanketops.dev labels", in.Labels)
	}
	if in.Source.RepositoryURL != "git@github.com:example-org/packages.git" || in.Source.CredentialsSecret != "packages-creds" {
		t.Errorf("source = %+v", in.Source)
	}
	if in.StateRepo.URL != "git@github.com:example-org/state.git" || in.StateRepo.Ref != "master" || in.StateRepo.Path != "./clusters/dev" {
		t.Errorf("state repo = %+v", in.StateRepo)
	}
}

func TestBuildPackageIntent_NoBlanketOpsLabels(t *testing.T) {
	in, err := BuildPackageIntent(newResolvedPackage(map[string]string{"app.kubernetes.io/name": "x"}))
	if err != nil {
		t.Fatalf("BuildPackageIntent: %v", err)
	}
	if in.Labels != nil {
		t.Errorf("labels = %v, want none", in.Labels)
	}
}

func TestBuildPackageIntent_RejectsWhatItCannotPlan(t *testing.T) {
	if _, err := BuildPackageIntent(nil); err == nil {
		t.Error("nil resolved package: want an error")
	}
	if _, err := BuildPackageIntent(&resolve.ResolvedPackage{}); err == nil {
		t.Error("resolved package without a spec: want an error")
	}
	disabled := newResolvedPackage(nil)
	disabled.Spec.Enabled = false
	if _, err := BuildPackageIntent(disabled); err == nil {
		t.Error("disabled package: want an error")
	}
}
