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

package api

import (
	"testing"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/build/domain"
	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

type createBuildSpecFunc func(domain.BuildSpec, *buildResolution.ResolvedBuild) (*shipwrightv1alpha1.Build, error)

// buildSpecProviders lists every provider's CreateBuildSpec so the trigger
// handling is asserted identically across backends.
func buildSpecProviders() map[string]createBuildSpecFunc {
	log := logr.Discard()
	return map[string]createBuildSpecFunc{
		"buildah":    NewBuildahProvider(nil, nil, log, nil).CreateBuildSpec,
		"kaniko":     NewKanikoProvider(nil, nil, log, nil).CreateBuildSpec,
		"buildpacks": NewBuildpacksProvider(nil, nil, log, nil).CreateBuildSpec,
	}
}

func newResolvedBuild(annotations map[string]string) *buildResolution.ResolvedBuild {
	return &buildResolution.ResolvedBuild{
		Build: &buildv1.Build{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", Annotations: annotations},
		},
	}
}

func newDomainSpec(image string) domain.BuildSpec {
	return domain.BuildSpec{
		SourceURL:    "https://github.com/acme/app.git",
		Revision:     "main",
		StrategyName: "strategy",
		StrategyKind: "ClusterBuildStrategy",
		Image:        image,
	}
}

func TestCreateBuildSpec_TriggerSHA(t *testing.T) {
	const sha = "1111111aaaaaaa"

	tests := []struct {
		name         string
		annotations  map[string]string
		image        string
		wantRevision string
		wantImage    string
	}{
		{
			name:         "no annotations keeps spec values",
			image:        "ghcr.io/acme/app:main",
			wantRevision: "main",
			wantImage:    "ghcr.io/acme/app:main",
		},
		{
			name:         "empty sha keeps spec values",
			annotations:  map[string]string{triggerSHAAnnotation: ""},
			image:        "ghcr.io/acme/app:main",
			wantRevision: "main",
			wantImage:    "ghcr.io/acme/app:main",
		},
		{
			name:         "sha replaces revision and tag",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "ghcr.io/acme/app:main",
			wantRevision: sha,
			wantImage:    "ghcr.io/acme/app:" + sha,
		},
		{
			name:         "sha is appended to an untagged image",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "ghcr.io/acme/app",
			wantRevision: sha,
			wantImage:    "ghcr.io/acme/app:" + sha,
		},
		{
			name:         "registry port is not mistaken for a tag",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "localhost:5000/app",
			wantRevision: sha,
			wantImage:    "localhost:5000/app:" + sha,
		},
	}

	for provider, create := range buildSpecProviders() {
		for _, tt := range tests {
			t.Run(provider+"/"+tt.name, func(t *testing.T) {
				got, err := create(newDomainSpec(tt.image), newResolvedBuild(tt.annotations))
				if err != nil {
					t.Fatalf("CreateBuildSpec: %v", err)
				}
				if got.Spec.Source.Revision == nil || *got.Spec.Source.Revision != tt.wantRevision {
					t.Errorf("revision = %v, want %q", got.Spec.Source.Revision, tt.wantRevision)
				}
				if got.Spec.Output.Image != tt.wantImage {
					t.Errorf("image = %q, want %q", got.Spec.Output.Image, tt.wantImage)
				}
			})
		}
	}
}

func TestCreateBuildSpec_RequiresSourceAndImage(t *testing.T) {
	for provider, create := range buildSpecProviders() {
		t.Run(provider, func(t *testing.T) {
			spec := newDomainSpec("ghcr.io/acme/app:main")
			spec.SourceURL = ""
			if _, err := create(spec, newResolvedBuild(nil)); err == nil {
				t.Error("expected an error for an empty source URL")
			}
			if _, err := create(newDomainSpec(""), newResolvedBuild(nil)); err == nil {
				t.Error("expected an error for an empty image")
			}
		})
	}
}
