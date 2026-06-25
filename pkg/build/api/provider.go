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

/*
This file owns the Provider interface — the execution contract that all build
backend implementations (Buildah, Kaniko, Buildpacks) must satisfy.

A Provider receives a resolved Build and a domain spec, dispatches a Shipwright
BuildRun, and returns immediately. It does not block on completion — the
buildrun observer writes the final outcome back to the Build CR asynchronously.

Concrete implementations live alongside this file (buildah.go, kaniko.go,
buildpacks.go). The BackendSelector (pkg/build/application/backend_selector.go)
routes a resolved BuildSpec to the correct provider based on the strategy name.
*/
package api

import (
	"context"

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
)

// Provider is the build execution contract implemented by all build backends.
//
//   - resolved.Build → Kubernetes object metadata, namespace, owner refs.
//   - resolved.Spec  → typed contract semantics decoded from spec.contract.
//   - spec           → internal domain projection for Shipwright spec construction.
type Provider interface {
	Run(ctx context.Context, resolved *buildResolution.ResolvedBuild, spec domain.BuildSpec) (domain.BuildResult, error)
}
