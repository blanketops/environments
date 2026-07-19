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
	"context"
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/blanketops/environments/cache/internal/testutil"
	"github.com/blanketops/environments/core/cache"
	serviceunitResolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

func newTestServiceUnitCache(t *testing.T) *ServiceUnitCache {
	t.Helper()
	return NewServiceUnitCache(&cache.Cache{External: testutil.NewFakeExternalCache()})
}

func TestServiceUnitCache_FieldRoundTrips(t *testing.T) {
	c := newTestServiceUnitCache(t)
	ctx := context.Background()
	nn := types.NamespacedName{Namespace: "default", Name: "api"}

	if err := c.SetType(ctx, nn, 1, "static"); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	if v, found, err := c.GetType(ctx, nn, 1); err != nil || !found || v != "static" {
		t.Errorf("GetType = %q, %v, %v; want static, true, nil", v, found, err)
	}

	if err := c.SetImage(ctx, nn, 1, "docker.io/x:v1"); err != nil {
		t.Fatalf("SetImage: %v", err)
	}
	if v, found, err := c.GetImage(ctx, nn, 1); err != nil || !found || v != "docker.io/x:v1" {
		t.Errorf("GetImage = %q, %v, %v", v, found, err)
	}

	buildRef := map[string]string{"name": "b1", "namespace": "ci"}
	if err := c.SetBuildRef(ctx, nn, 1, buildRef); err != nil {
		t.Fatalf("SetBuildRef: %v", err)
	}
	var gotRef map[string]string
	if found, err := c.GetBuildRef(ctx, nn, 1, &gotRef); err != nil || !found || gotRef["name"] != "b1" {
		t.Errorf("GetBuildRef = %v, %v, %v", gotRef, found, err)
	}

	if err := c.SetContainerPort(ctx, nn, 1, 8080); err != nil {
		t.Fatalf("SetContainerPort: %v", err)
	}
	if v, found, err := c.GetContainerPort(ctx, nn, 1); err != nil || !found || v != 8080 {
		t.Errorf("GetContainerPort = %d, %v, %v", v, found, err)
	}

	if err := c.SetSize(ctx, nn, 1, 3); err != nil {
		t.Fatalf("SetSize: %v", err)
	}
	if v, found, err := c.GetSize(ctx, nn, 1); err != nil || !found || v != 3 {
		t.Errorf("GetSize = %d, %v, %v", v, found, err)
	}

	if err := c.SetAppType(ctx, nn, 1, "web"); err != nil {
		t.Fatalf("SetAppType: %v", err)
	}
	if v, found, err := c.GetAppType(ctx, nn, 1); err != nil || !found || v != "web" {
		t.Errorf("GetAppType = %q, %v, %v", v, found, err)
	}

	if err := c.SetStackType(ctx, nn, 1, "nodejs"); err != nil {
		t.Fatalf("SetStackType: %v", err)
	}
	if v, found, err := c.GetStackType(ctx, nn, 1); err != nil || !found || v != "nodejs" {
		t.Errorf("GetStackType = %q, %v, %v", v, found, err)
	}
}

func TestServiceUnitCache_PublishResolved(t *testing.T) {
	nn := types.NamespacedName{Namespace: "default", Name: "api"}

	t.Run("nil resolved is a no-op", func(t *testing.T) {
		c := newTestServiceUnitCache(t)
		if err := c.PublishResolved(context.Background(), nn, 1, nil); err != nil {
			t.Fatalf("PublishResolved(nil): %v", err)
		}
	})

	t.Run("nil spec is a no-op", func(t *testing.T) {
		c := newTestServiceUnitCache(t)
		r := &serviceunitResolution.ResolvedServiceUnit{Spec: nil}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved(nil spec): %v", err)
		}
	})

	t.Run("static publishes image, not buildRef", func(t *testing.T) {
		c := newTestServiceUnitCache(t)
		r := &serviceunitResolution.ResolvedServiceUnit{
			Spec: &serviceunitResolution.ResolvedServiceUnitSpec{
				Type:          commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
				Image:         "docker.io/x:v1",
				ContainerPort: 8080,
				Size:          2,
				AppType:       "web",
				StackType:     "nodejs",
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}

		if v, found, _ := c.GetImage(context.Background(), nn, 1); !found || v != "docker.io/x:v1" {
			t.Errorf("GetImage = %q, %v; want docker.io/x:v1, true", v, found)
		}
		if _, found, _ := c.GetType(context.Background(), nn, 1); !found {
			t.Error("GetType: not found")
		}
		var buildRef map[string]any
		if found, _ := c.GetBuildRef(context.Background(), nn, 1, &buildRef); found {
			t.Error("GetBuildRef: found = true, want false for a STATIC unit")
		}
	})

	t.Run("build publishes buildRef, not image", func(t *testing.T) {
		c := newTestServiceUnitCache(t)
		r := &serviceunitResolution.ResolvedServiceUnit{
			Spec: &serviceunitResolution.ResolvedServiceUnitSpec{
				Type:     commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD,
				BuildRef: &serviceunitResolution.ResolvedBuildRef{Name: "b1", Namespace: "ci"},
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}

		var gotRef serviceunitResolution.ResolvedBuildRef
		if found, err := c.GetBuildRef(context.Background(), nn, 1, &gotRef); err != nil || !found || gotRef.Name != "b1" {
			t.Errorf("GetBuildRef = %+v, %v, %v", gotRef, found, err)
		}
		if v, found, _ := c.GetImage(context.Background(), nn, 1); found || v != "" {
			t.Errorf("GetImage = %q, %v; want unset for a BUILD unit", v, found)
		}
	})

	t.Run("zero ContainerPort and Size are not published", func(t *testing.T) {
		c := newTestServiceUnitCache(t)
		r := &serviceunitResolution.ResolvedServiceUnit{
			Spec: &serviceunitResolution.ResolvedServiceUnitSpec{
				Type:  commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_STATIC,
				Image: "docker.io/x:v1",
			},
		}
		if err := c.PublishResolved(context.Background(), nn, 1, r); err != nil {
			t.Fatalf("PublishResolved: %v", err)
		}
		if _, found, _ := c.GetContainerPort(context.Background(), nn, 1); found {
			t.Error("GetContainerPort: found = true, want false for a zero-value ContainerPort")
		}
		if _, found, _ := c.GetSize(context.Background(), nn, 1); found {
			t.Error("GetSize: found = true, want false for a zero-value Size")
		}
	})
}
