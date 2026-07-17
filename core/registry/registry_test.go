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

package registry

import (
	"context"
	"sort"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	command "github.com/blanketops/environments/core/command"
)

var buildGVK = schema.GroupVersionKind{Group: "environments.blanketops.dev", Version: "v1alpha1", Kind: "Build"}
var deploymentGVK = schema.GroupVersionKind{Group: "environments.blanketops.dev", Version: "v1alpha1", Kind: "Deployment"}

type fakeDomain struct {
	gvk schema.GroupVersionKind
	tag string
}

func (f *fakeDomain) GVK() schema.GroupVersionKind                  { return f.gvk }
func (f *fakeDomain) Handle(context.Context, command.Command) error { return nil }
func (f *fakeDomain) CanCreate(client.Object) bool                  { return true }
func (f *fakeDomain) CanUpdate(client.Object, client.Object) bool   { return true }
func (f *fakeDomain) CanDelete(client.Object) bool                  { return true }

func TestNewRegistry_StartsEmpty(t *testing.T) {
	r := NewRegistry()

	if _, ok := r.GetDomain(buildGVK); ok {
		t.Fatal("expected no domains registered on a new Registry")
	}
	if _, ok := r.GetStrategy("dockerfile"); ok {
		t.Fatal("expected no strategies registered on a new Registry")
	}
	if got := r.ListDomains(); len(got) != 0 {
		t.Fatalf("expected empty ListDomains, got %+v", got)
	}
	if got := r.ListStrategies(); len(got) != 0 {
		t.Fatalf("expected empty ListStrategies, got %+v", got)
	}
}

func TestRegistry_RegisterAndGetDomain(t *testing.T) {
	r := NewRegistry()
	d := &fakeDomain{gvk: buildGVK, tag: "first"}

	r.RegisterDomain(buildGVK, d)

	got, ok := r.GetDomain(buildGVK)
	if !ok {
		t.Fatal("expected to find the registered domain")
	}
	if got != d {
		t.Fatalf("expected GetDomain to return the exact registered instance, got %+v", got)
	}
}

func TestRegistry_RegisterDomain_OverwritesOnDuplicateGVK(t *testing.T) {
	r := NewRegistry()
	first := &fakeDomain{gvk: buildGVK, tag: "first"}
	second := &fakeDomain{gvk: buildGVK, tag: "second"}

	r.RegisterDomain(buildGVK, first)
	r.RegisterDomain(buildGVK, second)

	got, ok := r.GetDomain(buildGVK)
	if !ok {
		t.Fatal("expected to find a registered domain")
	}
	if got.(*fakeDomain).tag != "second" {
		t.Fatalf("expected the second registration to win, got tag %q", got.(*fakeDomain).tag)
	}
}

func TestRegistry_GetDomain_NotFound(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.GetDomain(buildGVK); ok {
		t.Fatal("expected GetDomain to report not-found for an unregistered GVK")
	}
}

func TestRegistry_RegisterAndGetStrategy(t *testing.T) {
	r := NewRegistry()
	r.RegisterStrategy("dockerfile", "docker-strategy-impl")

	got, ok := r.GetStrategy("dockerfile")
	if !ok {
		t.Fatal("expected to find the registered strategy")
	}
	if got != "docker-strategy-impl" {
		t.Fatalf("unexpected strategy value: %v", got)
	}
}

func TestRegistry_RegisterStrategy_OverwritesOnDuplicateName(t *testing.T) {
	r := NewRegistry()
	r.RegisterStrategy("dockerfile", "v1")
	r.RegisterStrategy("dockerfile", "v2")

	got, _ := r.GetStrategy("dockerfile")
	if got != "v2" {
		t.Fatalf("expected the second registration to win, got %v", got)
	}
}

func TestRegistry_GetStrategy_NotFound(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.GetStrategy("nonexistent"); ok {
		t.Fatal("expected GetStrategy to report not-found for an unregistered name")
	}
}

func TestRegistry_ListDomains(t *testing.T) {
	r := NewRegistry()
	r.RegisterDomain(buildGVK, &fakeDomain{gvk: buildGVK})
	r.RegisterDomain(deploymentGVK, &fakeDomain{gvk: deploymentGVK})

	got := r.ListDomains()
	if len(got) != 2 {
		t.Fatalf("expected 2 GVKs, got %d", len(got))
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Kind < got[j].Kind })
	if got[0] != buildGVK || got[1] != deploymentGVK {
		t.Fatalf("unexpected GVKs: %+v", got)
	}
}

func TestRegistry_ListStrategies(t *testing.T) {
	r := NewRegistry()
	r.RegisterStrategy("dockerfile", 1)
	r.RegisterStrategy("buildpacks-v3", 2)

	got := r.ListStrategies()
	if len(got) != 2 {
		t.Fatalf("expected 2 strategy names, got %d", len(got))
	}
	sort.Strings(got)
	if got[0] != "buildpacks-v3" || got[1] != "dockerfile" {
		t.Fatalf("unexpected strategy names: %+v", got)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			r.RegisterDomain(buildGVK, &fakeDomain{gvk: buildGVK})
			r.RegisterStrategy("dockerfile", i)
		}(i)
		go func() {
			defer wg.Done()
			r.GetDomain(buildGVK)
			r.GetStrategy("dockerfile")
			r.ListDomains()
			r.ListStrategies()
		}()
	}
	wg.Wait()
}
