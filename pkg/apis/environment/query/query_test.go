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

package query

import (
	"context"
	"testing"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	environmentapi "github.com/blanketops/environments/pkg/apis/environment/api"
)

const vaultContract = `{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","secretStore":{"provider":"vault","vault":{"address":"https://vault:8200","path":"secret","role":"environments"}}}`

func scheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := environmentsv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add environments-api to scheme: %v", err)
	}
	return s
}

func newEnv(name, namespace, contract string) *environmentsv1alpha1.Environment {
	return &environmentsv1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       environmentsv1alpha1.EnvironmentSpec{Contract: runtime.RawExtension{Raw: []byte(contract)}},
	}
}

func composedLabels(name, envType string) map[string]string {
	l := map[string]string{}
	if name != "" {
		l[LabelEnvironmentName] = name
	}
	if envType != "" {
		l[LabelEnvironmentType] = envType
	}
	return l
}

func TestLookup_MissingNameLabel_Errors(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(scheme(t)).Build()
	if _, err := Lookup(context.Background(), c, "readiness", composedLabels("", "dev")); err == nil {
		t.Fatal("expected error when the environment name label is absent")
	}
}

func TestLookup_EnvironmentNotFound_Errors(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(scheme(t)).Build()
	if _, err := Lookup(context.Background(), c, "readiness", composedLabels("ghost-env", "dev")); err == nil {
		t.Fatal("expected error when the named Environment does not exist (must pre-exist)")
	}
}

func TestLookup_Vault_DefaultsStoreNameAndKind(t *testing.T) {
	env := newEnv("readiness-env", "readiness", vaultContract)
	c := fake.NewClientBuilder().WithScheme(scheme(t)).WithObjects(env).Build()

	ctx, err := Lookup(context.Background(), c, "readiness", composedLabels("readiness-env", "dev"))
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ctx.Name != "readiness-env" {
		t.Fatalf("Name = %q, want readiness-env", ctx.Name)
	}
	if ctx.EnvironmentType != "dev" {
		t.Fatalf("EnvironmentType = %q (from label), want dev", ctx.EnvironmentType)
	}
	if ctx.StoreName != environmentapi.StoreVault {
		t.Fatalf("StoreName = %q, want %q (provider default)", ctx.StoreName, environmentapi.StoreVault)
	}
	if ctx.StoreKind != StoreKindClusterSecretStore {
		t.Fatalf("StoreKind = %q, want %q (platform default)", ctx.StoreKind, StoreKindClusterSecretStore)
	}
}

func TestLookup_ExplicitStoreNameWins(t *testing.T) {
	const contract = `{"applicationName":"app","branch":"main","gitOwner":"me","environmentType":"development","version":"1.0.0","secretStore":{"provider":"vault","storeName":"team-a-store","vault":{"address":"https://vault:8200","path":"secret","role":"environments"}}}`
	env := newEnv("readiness-env", "readiness", contract)
	c := fake.NewClientBuilder().WithScheme(scheme(t)).WithObjects(env).Build()

	ctx, err := Lookup(context.Background(), c, "readiness", composedLabels("readiness-env", "dev"))
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if ctx.StoreName != "team-a-store" {
		t.Fatalf("StoreName = %q, want team-a-store (explicit contract override)", ctx.StoreName)
	}
}
