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
	"context"
	"testing"

	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

type fakeProvider struct {
	runtime intent.Runtime
}

func (f *fakeProvider) Runtime() intent.Runtime       { return f.runtime }
func (f *fakeProvider) Supports(intent.Strategy) bool { return true }
func (f *fakeProvider) Execute(context.Context, *intent.DeploymentIntent) (*domain.DeploymentResult, error) {
	return nil, nil
}

func TestProviderRegistry_ResolveAndRegister(t *testing.T) {
	k8s := &fakeProvider{runtime: intent.RuntimeKubernetes}
	reg := NewProviderRegistry(k8s)

	got, err := reg.Resolve(intent.RuntimeKubernetes)
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if got != Provider(k8s) {
		t.Errorf("Resolve returned %+v, want the registered kubernetes provider", got)
	}

	if _, err := reg.Resolve(intent.RuntimeECS); err == nil {
		t.Fatal("expected an error resolving an unregistered runtime, got nil")
	}

	ecs := &fakeProvider{runtime: intent.RuntimeECS}
	reg.Register(ecs)
	got, err = reg.Resolve(intent.RuntimeECS)
	if err != nil {
		t.Fatalf("Resolve after Register: unexpected error: %v", err)
	}
	if got != Provider(ecs) {
		t.Errorf("Resolve after Register returned %+v, want the newly registered ecs provider", got)
	}
}
