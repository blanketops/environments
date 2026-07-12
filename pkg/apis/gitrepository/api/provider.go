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

	sourcesv1alpha1 "github.com/BlanketOps/environments-api/api/sources/v1alpha1"
	"github.com/BlanketOps/environments/pkg/apis/gitrepository/domain"
)

// Provider is the orchestration interface for GitRepository backend providers.
// Each implementation is responsible for ensuring the provider-specific
// resources that back a GitRepository CR.
//
// Current implementations:
//   - GitHubProvider (github.go) — upjet-github Crossplane provider
//
// The BackendSelector resolves the correct Provider from spec.contract.provider.
type Provider interface {
	Ensure(ctx context.Context, cr *sourcesv1alpha1.GitRepository, spec domain.GitRepository) (domain.Result, error)
}
