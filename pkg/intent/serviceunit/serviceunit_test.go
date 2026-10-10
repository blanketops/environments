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

package serviceunit

import (
	"testing"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	serviceunitresolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

func resolvedUnit(spec serviceunitresolution.ResolvedServiceUnitSpec) *serviceunitresolution.ResolvedServiceUnit {
	return &serviceunitresolution.ResolvedServiceUnit{
		ServiceUnit: &environmentsv1alpha1.ServiceUnit{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}},
		Spec:        &spec,
	}
}

// A BUILD ServiceUnit runs the image its Build pushed, pulled with the
// registry secret that Build declared. Both are injected before the intent
// is built and carried onto it.
func TestResolveServiceUnitIntent_BuildTypeCarriesImageAndPullSecret(t *testing.T) {
	in, err := ResolveServiceUnitIntent(resolvedUnit(serviceunitresolution.ResolvedServiceUnitSpec{
		Type:            commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
		Image:           "ghcr.io/example-org/api:main@sha256:1111",
		ImagePullSecret: "registry-credentials",
		ContainerPort:   8080,
		Size:            2,
	}))
	if err != nil {
		t.Fatalf("ResolveServiceUnitIntent: %v", err)
	}
	if in.Name != "api" || in.Image != "ghcr.io/example-org/api:main@sha256:1111" || in.ImagePullSecret != "registry-credentials" || in.Port != 8080 || in.Size != 2 {
		t.Errorf("intent = %+v", in)
	}
}

// Until its Build has pushed an image a BUILD ServiceUnit cannot be deployed.
func TestResolveServiceUnitIntent_BuildTypeWithoutImageIsNotReady(t *testing.T) {
	_, err := ResolveServiceUnitIntent(resolvedUnit(serviceunitresolution.ResolvedServiceUnitSpec{
		Type: commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
	}))
	if err == nil {
		t.Fatal("ResolveServiceUnitIntent = nil error, want build not ready")
	}
}

// A STATIC ServiceUnit names its own image and no pull secret.
func TestResolveServiceUnitIntent_Static(t *testing.T) {
	in, err := ResolveServiceUnitIntent(resolvedUnit(serviceunitresolution.ResolvedServiceUnitSpec{
		Type:  commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
		Image: "docker.io/example-org/api:v1",
	}))
	if err != nil {
		t.Fatalf("ResolveServiceUnitIntent: %v", err)
	}
	if in.Image != "docker.io/example-org/api:v1" || in.ImagePullSecret != "" {
		t.Errorf("intent = %+v, want the static image and no pull secret", in)
	}
}
