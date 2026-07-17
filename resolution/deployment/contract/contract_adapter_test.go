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

package contract

import (
	"testing"

	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"

	"github.com/blanketops/environments/resolution/deployment/resolve"
)

func TestToDeploymentContract_NilSpec(t *testing.T) {
	var s *resolve.ResolvedDeploymentSpec
	if got := ToDeploymentContract(s); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToDeploymentContract_NoManifestsRepo(t *testing.T) {
	s := &resolve.ResolvedDeploymentSpec{ServiceUnits: []string{"su1"}, Runtime: resolve.RuntimeKubernetes}
	out := ToDeploymentContract(s)
	if out.ManifestsRepo != nil {
		t.Fatal("expected nil ManifestsRepo when not set")
	}
	if out.Runtime.Runtime != commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KUBERNETES_CONTAINER {
		t.Fatalf("unexpected runtime: %v", out.Runtime.Runtime)
	}
}

func TestToDeploymentContract_WithManifestsRepo(t *testing.T) {
	s := &resolve.ResolvedDeploymentSpec{
		ServiceUnits: []string{"su1"},
		Runtime:      resolve.RuntimeKnative,
		ManifestsRepo: &resolve.ResolvedManifestsRepo{
			URL: "git@x", Ref: "main", Path: "/k8s", Strategy: "kustomize", CloneSecret: "sec",
		},
	}
	out := ToDeploymentContract(s)
	if out.ManifestsRepo == nil || out.ManifestsRepo.Url != "git@x" || out.ManifestsRepo.Ref != "main" {
		t.Fatalf("unexpected ManifestsRepo: %+v", out.ManifestsRepo)
	}
	if out.Runtime.Runtime != commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_KNATIVE_SERVICE {
		t.Fatalf("unexpected runtime: %v", out.Runtime.Runtime)
	}
}

func TestToContractRuntime_UnknownDefaultsToUnspecified(t *testing.T) {
	got := toContractRuntime(resolve.Runtime("bogus"))
	if got.Runtime != commoncontractv1.DeploymentRuntime_DEPLOYMENT_RUNTIME_UNSPECIFIED {
		t.Fatalf("expected UNSPECIFIED, got %v", got.Runtime)
	}
}

func TestToContractReconciliationStrategy_Kustomize(t *testing.T) {
	got := toContractReconciliationStrategy(resolve.ReconciliationKustomize)
	if got.Strategy != commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_KUSTOMIZE {
		t.Fatalf("unexpected strategy: %v", got.Strategy)
	}
}

func TestToContractReconciliationStrategy_Helm(t *testing.T) {
	got := toContractReconciliationStrategy(resolve.ReconciliationHelm)
	if got.Strategy != commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_HELM {
		t.Fatalf("unexpected strategy: %v", got.Strategy)
	}
}

func TestToContractReconciliationStrategy_ImperativeDefaultsToUnspecified(t *testing.T) {
	got := toContractReconciliationStrategy(resolve.ReconciliationImperative)
	if got.Strategy != commoncontractv1.ReconciliationStrategy_RECONCILIATION_STRATEGY_UNSPECIFIED {
		t.Fatalf("expected UNSPECIFIED, got %v", got.Strategy)
	}
}
