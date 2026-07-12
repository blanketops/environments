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

package application

import (
	"context"

	"github.com/BlanketOps/environments/pkg/apis/deployment/api"
	"github.com/BlanketOps/environments/pkg/apis/deployment/domain"
	intent "github.com/BlanketOps/environments/pkg/intent/deployment"
)

type GitOpsDecorator struct {
	inner api.Provider
}

func NewGitOpsDecorator(inner api.Provider) api.Provider {
	return &GitOpsDecorator{
		inner: inner,
	}
}

func (g *GitOpsDecorator) Runtime() intent.Runtime {
	return g.inner.Runtime()
}

func (g *GitOpsDecorator) Supports(s intent.Strategy) bool {
	return g.inner.Supports(s)
}

func (g *GitOpsDecorator) Execute(
	ctx context.Context,
	i *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	// Commit manifests to repo / trigger Flux here

	return g.inner.Execute(ctx, i)
}
