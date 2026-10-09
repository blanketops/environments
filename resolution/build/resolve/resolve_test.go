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

package resolve

import (
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
	if resolved.Spec.Policy == nil {
		t.Fatal("expected a non-nil Policy when none is declared")
	}
	if len(resolved.Spec.Policy.Triggers) != 0 || resolved.Spec.Policy.Retry != nil {
		t.Fatalf("expected an empty Policy when none is declared, got %+v", resolved.Spec.Policy)
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
			"allowedTriggers":[{"type":"push"},{"type":"pull_request"}, "not-an-object"],
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

// TestResolveBuild_PolicyIsOptional covers every way a contract can leave
// the policy or its allowedTriggers out. Each must resolve, and to a policy
// consumers can read without a nil check.
func TestResolveBuild_PolicyIsOptional(t *testing.T) {
	tests := []struct {
		name         string
		policy       string
		wantTriggers int
		wantRetry    bool
	}{
		{name: "no policy key", policy: ""},
		{name: "null policy", policy: `,"policy":null`},
		{name: "policy of the wrong type", policy: `,"policy":"push"`},
		{name: "empty policy", policy: `,"policy":{}`},
		{name: "retry without allowedTriggers", policy: `,"policy":{"retry":{"onFailure":true,"maxAttempts":2}}`, wantRetry: true},
		{name: "null allowedTriggers", policy: `,"policy":{"allowedTriggers":null}`},
		{name: "empty allowedTriggers", policy: `,"policy":{"allowedTriggers":[]}`},
		{name: "allowedTriggers without retry", policy: `,"policy":{"allowedTriggers":[{"type":"push"}]}`, wantTriggers: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := buildWithContract(t, `{"image":"foo","source":{"url":"x"}`+tt.policy+`}`)
			resolved, err := ResolveBuild(b)
			if err != nil {
				t.Fatalf("ResolveBuild: %v", err)
			}
			if resolved.Spec.Policy == nil {
				t.Fatal("Policy is nil")
			}
			if got := len(resolved.Spec.Policy.Triggers); got != tt.wantTriggers {
				t.Errorf("triggers = %d, want %d", got, tt.wantTriggers)
			}
			if got := resolved.Spec.Policy.Retry != nil; got != tt.wantRetry {
				t.Errorf("retry present = %v, want %v", got, tt.wantRetry)
			}
		})
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

func FuzzResolveBuild(f *testing.F) {
	f.Add(`{"image":"foo:latest","source":{"url":"git@github.com:x/y"}}`)
	f.Add(`{not json`)
	f.Add(`{"image":"foo","source":{}}`)
	f.Add(`{"image":"foo","source":{"url":"git@x","cloneSecret":""}}`)
	f.Add(`{"image":"foo","source":{"url":"x"},"policy":{"retry":{"onFailure":true,"maxAttempts":0}}}`)
	f.Add(`{"image":"foo","source":{"url":"x"},"serviceAccount":{"name":"sa","secret":"sa-secret"}}`)
	f.Add(`{"image":"foo","source":{"url":"x"},"strategy":{"kind":"BogusStrategy"}}`)

	f.Fuzz(func(t *testing.T, raw string) {
		b := buildWithContract(t, raw)
		_, _ = ResolveBuild(b)
	})
}
