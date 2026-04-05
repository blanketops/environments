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

package resolution

import (
	"context"
	"fmt"

	build "github.com/ntlaletsi70/blanketops-environments/resolution/build"
	"github.com/ntlaletsi70/blanketops-environments/resolution/buildtrigger"
	deployment "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	gitHubEvent "github.com/ntlaletsi70/blanketops-environments/resolution/githubevent"
	gitRepository "github.com/ntlaletsi70/blanketops-environments/resolution/gitrepository"

	events1 "github.com/ntlaletsi70/blanketops-environments-api/api/events/v1alpha1"
	sourcesv1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/sources/v1alpha1"

	env1alpha1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Adapter struct {
	build         *build.Adapter
	buildtrigger  *buildtrigger.Adapter
	deployment    *deployment.Adapter
	gitrepository *gitRepository.Adapter
	githubevent   *gitHubEvent.Adapter
}

func NewAdapter() *Adapter {
	return &Adapter{
		build:         build.NewAdapter(),
		deployment:    deployment.NewAdapter(),
		gitrepository: gitRepository.NewAdapter(),
		githubevent:   gitHubEvent.NewAdapter(),
	}
}

func (a *Adapter) Resolve(ctx context.Context, obj client.Object) error {
	switch o := obj.(type) {

	case *env1alpha1.Build:
		_, err := a.build.Resolve(ctx, o)
		return err

	case *env1alpha1.Deployment:
		_, err := a.deployment.Resolve(ctx, o)
		return err

	case *sourcesv1alpha1.GitRepository:
		_, err := a.gitrepository.Resolve(ctx, o)
		return err
	case *env1alpha1.BuildTrigger:
		_, err := a.buildtrigger.Resolve(ctx, o)
		return err
	case *events1.GitHubEvent:
		_, err := a.githubevent.Resolve(ctx, o)
		return err
	default:
		return fmt.Errorf(
			"unsupported object type %T in resolution adapter",
			obj,
		)
	}
}
