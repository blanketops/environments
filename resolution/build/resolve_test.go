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

package build

import (
	"context"
	"strings"
	"testing"

	environmentv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func buildWithContract(t *testing.T, raw string) *environmentv1alpha1.Build {
	t.Helper()
	return &environmentv1alpha1.Build{
		Spec: environmentv1alpha1.BuildSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

func TestResolveBuild_NilBuild(t *testing.T) {
	if _, err := ResolveBuild(nil); err == nil {
		t.Fatal("expected error for nil build")
	}
}

func TestResolveBuild_EmptyContract(t *testing.T) {
	b := &environmentv1alpha1.Build{}
	if _, err := ResolveBuild(b); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveBuild_InvalidJSON(t *testing.T) {
	b := buildWithContract(t, `{not json`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveBuild_MissingSource(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo"}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "source is required") {
		t.Fatalf("expected source-required error, got %v", err)
	}
}

func TestResolveBuild_MissingSourceURL(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{}}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "source.url") {
		t.Fatalf("expected source.url error, got %v", err)
	}
}

func TestResolveBuild_CloneSecretDeclaredButEmpty(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"git@x","cloneSecret":""}}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "cloneSecret declared but resolved empty") {
		t.Fatalf("expected cloneSecret error, got %v", err)
	}
}

func TestResolveBuild_MissingImage(t *testing.T) {
	b := buildWithContract(t, `{"source":{"url":"git@x"}}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("expected image error, got %v", err)
	}
}

func TestResolveBuild_MinimalValid(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo:latest","source":{"url":"git@github.com:x/y"}}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if resolved.Spec.Image != "foo:latest" {
		t.Fatalf("unexpected image: %q", resolved.Spec.Image)
	}
	if resolved.Spec.Source.URL != "git@github.com:x/y" {
		t.Fatalf("unexpected source url: %q", resolved.Spec.Source.URL)
	}
	if resolved.Spec.ServiceAccount != nil {
		t.Fatal("expected nil ServiceAccount when not declared")
	}
	if resolved.Spec.Policy != nil {
		t.Fatal("expected nil Policy when not declared")
	}
	if resolved.Build != b {
		t.Fatal("expected resolved.Build to reference the original CR")
	}
}

func TestResolveBuild_StrategyKindClusterBuildStrategy(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"strategy":{"name":"buildpacks","kind":"ClusterBuildStrategy"}}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if resolved.Spec.Strategy.Name != "buildpacks" || resolved.Spec.Strategy.StrategyKind != "ClusterBuildStrategy" {
		t.Fatalf("unexpected strategy: %+v", resolved.Spec.Strategy)
	}
}

func TestResolveBuild_StrategyKindNamespacedBuildStrategy(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"strategy":{"kind":"NamespacedBuildStrategy"}}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if resolved.Spec.Strategy.StrategyKind != "NamespacedBuildStrategy" {
		t.Fatalf("unexpected strategy kind: %q", resolved.Spec.Strategy.StrategyKind)
	}
}

func TestResolveBuild_StrategyKindUnsupported(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"strategy":{"kind":"BogusStrategy"}}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "unsupported strategy.kind") {
		t.Fatalf("expected unsupported strategy.kind error, got %v", err)
	}
}

func TestResolveBuild_ServiceAccount(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"serviceAccount":{"name":"sa","secret":"sa-secret"}}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if resolved.Spec.ServiceAccount == nil || resolved.Spec.ServiceAccount.Name != "sa" || resolved.Spec.ServiceAccount.Secret != "sa-secret" {
		t.Fatalf("unexpected service account: %+v", resolved.Spec.ServiceAccount)
	}
}

func TestResolveBuild_PolicyTriggersAndRetry(t *testing.T) {
	b := buildWithContract(t, `{
		"image":"foo","source":{"url":"x"},
		"policy":{
			"triggers":[{"type":"push"},{"type":"pull_request"}, "not-an-object"],
			"retry":{"onFailure":true,"maxAttempts":3}
		}
	}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if len(resolved.Spec.Policy.Triggers) != 2 {
		t.Fatalf("expected 2 valid triggers (non-object entries skipped), got %d: %+v",
			len(resolved.Spec.Policy.Triggers), resolved.Spec.Policy.Triggers)
	}
	if resolved.Spec.Policy.Retry == nil || !resolved.Spec.Policy.Retry.OnFailure || resolved.Spec.Policy.Retry.MaxAttempts != 3 {
		t.Fatalf("unexpected retry policy: %+v", resolved.Spec.Policy.Retry)
	}
}

func TestResolveBuild_PolicyRetryOnFailureRequiresMaxAttempts(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"policy":{"retry":{"onFailure":true,"maxAttempts":0}}}`)
	_, err := ResolveBuild(b)
	if err == nil || !strings.Contains(err.Error(), "maxAttempts must be > 0") {
		t.Fatalf("expected maxAttempts error, got %v", err)
	}
}

func TestResolveBuild_PolicyRetryOnFailureFalseAllowsZeroMaxAttempts(t *testing.T) {
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"},"policy":{"retry":{"onFailure":false,"maxAttempts":0}}}`)
	resolved, err := ResolveBuild(b)
	if err != nil {
		t.Fatalf("ResolveBuild: %v", err)
	}
	if resolved.Spec.Policy.Retry.OnFailure {
		t.Fatal("expected OnFailure to be false")
	}
}

func TestOptionalUint32_IntBranch(t *testing.T) {
	// json.Unmarshal into map[string]any never produces a plain int (always
	// float64), so this branch is unreachable via ResolveBuild — exercised
	// directly here for coverage of the helper itself.
	got := optionalUint32(map[string]any{"n": int(7)}, "n")
	if got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestOptionalUint32_Missing(t *testing.T) {
	if got := optionalUint32(map[string]any{}, "n"); got != 0 {
		t.Fatalf("expected 0 for missing key, got %d", got)
	}
}

func TestMustString_WrongType(t *testing.T) {
	_, err := mustString(map[string]any{"k": 5}, "k")
	if err == nil || !strings.Contains(err.Error(), "must be a non-empty string") {
		t.Fatalf("expected type error, got %v", err)
	}
}

func TestOptionalBool_WrongType(t *testing.T) {
	if got := optionalBool(map[string]any{"k": "not-a-bool"}, "k"); got != false {
		t.Fatalf("expected false for wrong type, got %v", got)
	}
}

func TestAdapter_Resolve(t *testing.T) {
	a := NewAdapter()
	b := buildWithContract(t, `{"image":"foo","source":{"url":"x"}}`)
	resolved, err := a.Resolve(context.Background(), b)
	if err != nil {
		t.Fatalf("Adapter.Resolve: %v", err)
	}
	if resolved.Spec.Image != "foo" {
		t.Fatalf("unexpected image: %q", resolved.Spec.Image)
	}
}
