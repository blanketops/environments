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

package cache

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"

	corecache "github.com/blanketops/environments/core/cache"

	"github.com/blanketops/environments/cache/internal/testutil"
)

func TestNewObjectCache_DefaultsZeroTTL(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	c := &corecache.Cache{External: fake}
	oc := NewObjectCache(c, "widget", 0)

	nn := types.NamespacedName{Namespace: "default", Name: "w1"}
	if err := oc.SetField(context.Background(), nn, 1, "field", "value"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	ttl, ok := fake.TTL("widget:default/w1:gen1:field")
	if !ok {
		t.Fatal("key not found in fake cache")
	}
	if ttl != 1*time.Hour {
		t.Errorf("TTL = %v, want default 1h when constructed with ttl<=0", ttl)
	}
}

func TestNewObjectCache_PreservesExplicitTTL(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	c := &corecache.Cache{External: fake}
	oc := NewObjectCache(c, "widget", 5*time.Minute)

	nn := types.NamespacedName{Namespace: "default", Name: "w1"}
	if err := oc.SetField(context.Background(), nn, 1, "field", "value"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	ttl, _ := fake.TTL("widget:default/w1:gen1:field")
	if ttl != 5*time.Minute {
		t.Errorf("TTL = %v, want 5m", ttl)
	}
}

func TestObjectCache_KeyFormat(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	oc := NewObjectCache(&corecache.Cache{External: fake}, "build", time.Hour)
	nn := types.NamespacedName{Namespace: "ci", Name: "worker"}

	if err := oc.SetField(context.Background(), nn, 42, "image", "docker.io/x:v1"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	keys := fake.Keys()
	if len(keys) != 1 || keys[0] != "build:ci/worker:gen42:image" {
		t.Errorf("keys = %v, want exactly [build:ci/worker:gen42:image]", keys)
	}
}

func TestObjectCache_SetGet_RoundTrip(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	oc := NewObjectCache(&corecache.Cache{External: fake}, "build", time.Hour)
	nn := types.NamespacedName{Namespace: "default", Name: "w1"}

	if err := oc.SetField(context.Background(), nn, 1, "image", "docker.io/x:v1"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	var got string
	found, err := oc.GetField(context.Background(), nn, 1, "image", &got)
	if err != nil {
		t.Fatalf("GetField: %v", err)
	}
	if !found {
		t.Fatal("GetField: found = false, want true")
	}
	if got != "docker.io/x:v1" {
		t.Errorf("got = %q, want docker.io/x:v1", got)
	}
}

func TestObjectCache_GetField_Miss(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	oc := NewObjectCache(&corecache.Cache{External: fake}, "build", time.Hour)
	nn := types.NamespacedName{Namespace: "default", Name: "missing"}

	var got string
	found, err := oc.GetField(context.Background(), nn, 1, "image", &got)
	if err != nil {
		t.Fatalf("GetField: %v", err)
	}
	if found {
		t.Fatal("GetField: found = true, want false for a never-set field")
	}
}

func TestObjectCache_GenerationScoping(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	oc := NewObjectCache(&corecache.Cache{External: fake}, "build", time.Hour)
	nn := types.NamespacedName{Namespace: "default", Name: "w1"}

	if err := oc.SetField(context.Background(), nn, 1, "image", "v1"); err != nil {
		t.Fatalf("SetField gen1: %v", err)
	}
	if err := oc.SetField(context.Background(), nn, 2, "image", "v2"); err != nil {
		t.Fatalf("SetField gen2: %v", err)
	}

	var gen1, gen2 string
	if _, err := oc.GetField(context.Background(), nn, 1, "image", &gen1); err != nil {
		t.Fatalf("GetField gen1: %v", err)
	}
	if _, err := oc.GetField(context.Background(), nn, 2, "image", &gen2); err != nil {
		t.Fatalf("GetField gen2: %v", err)
	}

	if gen1 != "v1" || gen2 != "v2" {
		t.Errorf("gen1=%q gen2=%q, want v1/v2 — a spec change must not clobber the prior generation's entry", gen1, gen2)
	}
}

func TestObjectCache_Invalidate_DropsAllFieldsAndGenerations(t *testing.T) {
	fake := testutil.NewFakeExternalCache()
	oc := NewObjectCache(&corecache.Cache{External: fake}, "build", time.Hour)
	nn := types.NamespacedName{Namespace: "default", Name: "w1"}
	other := types.NamespacedName{Namespace: "default", Name: "w2"}

	if err := oc.SetField(context.Background(), nn, 1, "image", "v1"); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	if err := oc.SetField(context.Background(), nn, 2, "strategy", "kaniko"); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	if err := oc.SetField(context.Background(), other, 1, "image", "unrelated"); err != nil {
		t.Fatalf("SetField other: %v", err)
	}

	if err := oc.Invalidate(context.Background(), nn); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	remaining := fake.Keys()
	if len(remaining) != 1 || remaining[0] != "build:default/w2:gen1:image" {
		t.Errorf("remaining keys = %v, want only the other object's key surviving", remaining)
	}
}
