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

	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
	buildResolution "github.com/ntlaletsi70/blanketops-environments/resolution/build"
)

type Provider interface {
	// Run executes a build using a resolved build and pure domain spec.
	//
	// - resolved.Build → metadata / owner refs / namespace
	// - resolved.Spec  → typed contract semantics
	// - spec           → internal domain projection
	Run(
		ctx context.Context,
		resolved *buildResolution.ResolvedBuild,
		spec domain.BuildSpec,
	) (domain.BuildResult, error)
}
