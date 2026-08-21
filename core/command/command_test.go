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

package command

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCommand_Name_NilObj(t *testing.T) {
	var c Command
	if got := c.Name(); got != "" {
		t.Fatalf("expected empty name for nil Obj, got %q", got)
	}
}

func TestCommand_Name_PopulatedObj(t *testing.T) {
	c := Command{Obj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}}}
	if got := c.Name(); got != "my-pod" {
		t.Fatalf("expected %q, got %q", "my-pod", got)
	}
}

func TestCommand_Namespace_NilObj(t *testing.T) {
	var c Command
	if got := c.Namespace(); got != "" {
		t.Fatalf("expected empty namespace for nil Obj, got %q", got)
	}
}

func TestCommand_Namespace_PopulatedObj(t *testing.T) {
	c := Command{Obj: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"}}}
	if got := c.Namespace(); got != "my-ns" {
		t.Fatalf("expected %q, got %q", "my-ns", got)
	}
}

func TestCommand_String(t *testing.T) {
	c := Command{
		Type: CmdUpdate,
		GVK:  schema.GroupVersionKind{Group: "environments.blanketops.dev", Version: "v1alpha1", Kind: "Build"},
		Obj:  &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-build", Namespace: "default"}},
	}
	got := c.String()
	for _, want := range []string{string(CmdUpdate), "default", "my-build", "Build"} {
		if !strings.Contains(got, want) {
			t.Fatalf("String() = %q, expected it to contain %q", got, want)
		}
	}
}

func TestCommand_Clone_DeepCopiesObj(t *testing.T) {
	original := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}}
	c := Command{Type: CmdCreate, Obj: original}

	clone := c.Clone()

	if clone.Obj == original {
		t.Fatal("expected Clone to deep-copy Obj, but it shares the same pointer")
	}
	if clone.Obj.GetName() != "my-pod" {
		t.Fatalf("expected cloned Obj to have the same name, got %q", clone.Obj.GetName())
	}

	// Mutating the original must not affect the clone.
	original.SetName("mutated")
	if clone.Obj.GetName() != "my-pod" {
		t.Fatalf("expected clone to be unaffected by mutation of the original, got %q", clone.Obj.GetName())
	}
}

func TestCommand_Clone_PreservesOldAndNewByReference(t *testing.T) {
	oldObj := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "old"}}
	newObj := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "new"}}
	c := Command{Type: CmdUpdate, Obj: newObj, Old: oldObj, New: newObj}

	clone := c.Clone()

	if clone.Old != oldObj {
		t.Fatal("expected Old to be reference-copied, not deep-copied")
	}
	if clone.New != newObj {
		t.Fatal("expected New to be reference-copied, not deep-copied")
	}
}

func TestCommand_Clone_NilObjDoesNotPanic(t *testing.T) {
	c := Command{Type: CmdCreate}

	clone := c.Clone()

	if clone.Obj != nil {
		t.Fatalf("expected clone.Obj to stay nil, got %v", clone.Obj)
	}
}
