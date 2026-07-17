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

func deploymentWithContract(raw string) *environmentv1alpha1.Deployment {
	return &environmentv1alpha1.Deployment{
		Spec: environmentv1alpha1.DeploymentSpec{
			Contract: runtime.RawExtension{Raw: []byte(raw)},
		},
	}
}

const minimalValid = `{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling"}`

func TestResolveDeployment_NilDeployment(t *testing.T) {
	if _, err := ResolveDeployment(nil); err == nil {
		t.Fatal("expected error for nil deployment")
	}
}

func TestResolveDeployment_EmptyContract(t *testing.T) {
	d := &environmentv1alpha1.Deployment{}
	if _, err := ResolveDeployment(d); err == nil {
		t.Fatal("expected error for empty contract")
	}
}

func TestResolveDeployment_InvalidJSON(t *testing.T) {
	d := deploymentWithContract(`{not json`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "failed to decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestResolveDeployment_ServiceUnitsMissing(t *testing.T) {
	d := deploymentWithContract(`{"runtime":"kubernetes","strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "serviceUnits is required") {
		t.Fatalf("expected serviceUnits error, got %v", err)
	}
}

func TestResolveDeployment_ServiceUnitsEmpty(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":[],"runtime":"kubernetes","strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "serviceUnits is required") {
		t.Fatalf("expected serviceUnits error, got %v", err)
	}
}

func TestResolveDeployment_ServiceUnitsEntryNotString(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":[1],"runtime":"kubernetes","strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "must be non-empty strings") {
		t.Fatalf("expected entries error, got %v", err)
	}
}

func TestResolveDeployment_RuntimeMissing(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "runtime is required") {
		t.Fatalf("expected runtime-required error, got %v", err)
	}
}

func TestResolveDeployment_RuntimeUnsupportedString(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"bogus","strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "unsupported deployment.runtime") {
		t.Fatalf("expected unsupported runtime error, got %v", err)
	}
}

func TestResolveDeployment_RuntimeKnative(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"knative","strategy":"Rolling"}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.Runtime != RuntimeKnative {
		t.Fatalf("expected RuntimeKnative, got %q", resolved.Spec.Runtime)
	}
}

func TestResolveDeployment_RuntimeRawEnumValue(t *testing.T) {
	// float64(1) == DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":1,"strategy":"Rolling"}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.Runtime != RuntimeKubernetes {
		t.Fatalf("expected RuntimeKubernetes, got %q", resolved.Spec.Runtime)
	}
}

func TestResolveDeployment_RuntimeRawEnumValueUnsupported(t *testing.T) {
	// float64(99) has no corresponding domain Runtime.
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":99,"strategy":"Rolling"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "unsupported deployment runtime enum") {
		t.Fatalf("expected unsupported enum error, got %v", err)
	}
}

func TestResolveDeployment_StrategyMissing(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "deployment.strategy is required") {
		t.Fatalf("expected strategy-required error, got %v", err)
	}
}

func TestResolveDeployment_StrategyBlueGreenAndCanary(t *testing.T) {
	for strategy, want := range map[string]Strategy{"BlueGreen": StrategyBlueGreen, "Canary": StrategyCanary} {
		d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"` + strategy + `"}`)
		resolved, err := ResolveDeployment(d)
		if err != nil {
			t.Fatalf("ResolveDeployment(%s): %v", strategy, err)
		}
		if resolved.Spec.Strategy != want {
			t.Fatalf("expected %q, got %q", want, resolved.Spec.Strategy)
		}
	}
}

func TestResolveDeployment_StrategyUnsupported(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Bogus"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "unsupported deployment.strategy") {
		t.Fatalf("expected unsupported strategy error, got %v", err)
	}
}

func TestResolveDeployment_MinimalValid(t *testing.T) {
	d := deploymentWithContract(minimalValid)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if len(resolved.Spec.ServiceUnits) != 1 || resolved.Spec.ServiceUnits[0] != "su1" {
		t.Fatalf("unexpected ServiceUnits: %+v", resolved.Spec.ServiceUnits)
	}
	if resolved.Spec.Strategy != StrategyRolling {
		t.Fatalf("unexpected Strategy: %q", resolved.Spec.Strategy)
	}
	if resolved.Spec.ManifestsRepo != nil {
		t.Fatal("expected nil ManifestsRepo when not declared")
	}
	if resolved.Spec.ReconciliationStrategy != ReconciliationImperative {
		t.Fatalf("expected default Imperative reconciliation strategy, got %q", resolved.Spec.ReconciliationStrategy)
	}
}

func TestResolveDeployment_GitOwnerDeclaredButEmpty(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","gitOwner":""}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "gitOwner declared but empty") {
		t.Fatalf("expected gitOwner error, got %v", err)
	}
}

func TestResolveDeployment_ManifestsRepoMissingURL(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{},"reconciliationStrategy":"kustomize"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "manifestsRepo.url") {
		t.Fatalf("expected manifestsRepo.url error, got %v", err)
	}
}

func TestResolveDeployment_ManifestsRepoCloneSecretDeclaredButEmpty(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x","cloneSecret":""},"reconciliationStrategy":"kustomize"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "manifestsRepo.cloneSecret declared but empty") {
		t.Fatalf("expected cloneSecret error, got %v", err)
	}
}

func TestResolveDeployment_ManifestsRepoRequiresReconciliationStrategy(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"}}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "reconciliationStrategy is required when manifestsRepo is defined") {
		t.Fatalf("expected reconciliationStrategy-required error, got %v", err)
	}
}

func TestResolveDeployment_ReconciliationStrategyForbiddenWithoutManifestsRepo(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","reconciliationStrategy":"kustomize"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "reconciliationStrategy cannot be set when manifestsRepo is not defined") {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestResolveDeployment_ReconciliationStrategyKustomize(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x","ref":"main","path":"/k8s"},"reconciliationStrategy":"kustomize"}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.ReconciliationStrategy != ReconciliationKustomize {
		t.Fatalf("expected ReconciliationKustomize, got %q", resolved.Spec.ReconciliationStrategy)
	}
	if resolved.Spec.ManifestsRepo.Ref != "main" || resolved.Spec.ManifestsRepo.Path != "/k8s" {
		t.Fatalf("unexpected ManifestsRepo: %+v", resolved.Spec.ManifestsRepo)
	}
}

func TestResolveDeployment_ReconciliationStrategyHelm(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":"helm"}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.ReconciliationStrategy != ReconciliationHelm {
		t.Fatalf("expected ReconciliationHelm, got %q", resolved.Spec.ReconciliationStrategy)
	}
}

func TestResolveDeployment_ReconciliationStrategyUnsupportedString(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":"bogus"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "unsupported reconciliationStrategy") {
		t.Fatalf("expected unsupported reconciliationStrategy error, got %v", err)
	}
}

func TestResolveDeployment_ReconciliationStrategyRawEnumValue(t *testing.T) {
	// float64(1) == RECONCILIATION_STRATEGY_KUSTOMIZE
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":1}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.ReconciliationStrategy != ReconciliationKustomize {
		t.Fatalf("expected ReconciliationKustomize, got %q", resolved.Spec.ReconciliationStrategy)
	}
}

func TestResolveDeployment_ImageAutomation(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","imageAutomation":true}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if !resolved.Spec.ImageAutomation {
		t.Fatal("expected ImageAutomation to be true")
	}
}

func TestResolveDeployment_ReconciliationStrategyRawEnumUnspecified(t *testing.T) {
	// float64(0) == RECONCILIATION_STRATEGY_UNSPECIFIED
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":0}`)
	resolved, err := ResolveDeployment(d)
	if err != nil {
		t.Fatalf("ResolveDeployment: %v", err)
	}
	if resolved.Spec.ReconciliationStrategy != ReconciliationImperative {
		t.Fatalf("expected ReconciliationImperative for UNSPECIFIED enum, got %q", resolved.Spec.ReconciliationStrategy)
	}
}

func TestResolveDeployment_ReconciliationStrategyWrongType(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":true}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "reconciliationStrategy is required") {
		t.Fatalf("expected reconciliationStrategy-required error for wrong type, got %v", err)
	}
}

func TestResolveDeployment_StrategyWrongType(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":5}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "deployment.strategy is required") {
		t.Fatalf("expected deployment.strategy-required error for wrong type, got %v", err)
	}
}

func TestResolveDeployment_ManifestsRepoURLWrongType(t *testing.T) {
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":5},"reconciliationStrategy":"kustomize"}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "manifestsRepo.url") {
		t.Fatalf("expected manifestsRepo.url type error, got %v", err)
	}
}

func TestResolveDeployment_ReconciliationStrategyRawEnumOutOfRange(t *testing.T) {
	// resolveReconciliationStrategy passes any float64 straight through as an
	// enum value with no range validation — 99 has no corresponding domain
	// ReconciliationStrategy, exercising normalizeReconciliationStrategy's
	// default branch.
	d := deploymentWithContract(`{"serviceUnits":["su1"],"runtime":"kubernetes","strategy":"Rolling","manifestsRepo":{"url":"git@x"},"reconciliationStrategy":99}`)
	_, err := ResolveDeployment(d)
	if err == nil || !strings.Contains(err.Error(), "unsupported reconciliation strategy enum") {
		t.Fatalf("expected unsupported reconciliation strategy enum error, got %v", err)
	}
}
