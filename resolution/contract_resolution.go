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

	"sigs.k8s.io/controller-runtime/pkg/client"

	environments1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	sourcesv1alpha1 "github.com/blanketops/environments-api/api/sources/v1alpha1"
	//networksv1alpha1 "github.com/blanketops/environments-api/api/networks/v1alpha1"
	build "github.com/blanketops/environments/resolution/build/adapter"
	deployment "github.com/blanketops/environments/resolution/deployment/adapter"
	gitHubEvent "github.com/blanketops/environments/resolution/githubevent/adapter"
	gitRepository "github.com/blanketops/environments/resolution/gitrepository/adapter"
	packages "github.com/blanketops/environments/resolution/packages/adapter"
	serviceunit "github.com/blanketops/environments/resolution/serviceunit/adapter"
	// route "github.com/blanketops/environments/resolution/route/adapter"
	// domain "github.com/blanketops/environments/resolution/domain/adapter"
)

// Adapter dispatches resolution for any supported CR type to that type's
// own resolution adapter, based on the object's concrete Go type.
type Adapter struct {
	build         *build.Adapter
	deployment    *deployment.Adapter
	gitrepository *gitRepository.Adapter
	githubevent   *gitHubEvent.Adapter
	packages      *packages.Adapter
	serviceunit   *serviceunit.Adapter
	// domain        *domain.Adapter
	// route         *route.Adapter
}

// NewAdapter constructs an Adapter wired up with every supported CR type's
// own resolution adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		build:         build.NewAdapter(),
		deployment:    deployment.NewAdapter(),
		gitrepository: gitRepository.NewAdapter(),
		githubevent:   gitHubEvent.NewAdapter(),
		packages:      packages.NewAdapter(),
		serviceunit:   serviceunit.NewAdapter(),
		// domain:          domain.NewAdapter(),
		// route:           route.NewAdapter(),
	}
}

// Resolve dispatches obj to the resolution adapter matching its concrete
// type. Route and Domain are not yet wired up.
func (a *Adapter) Resolve(ctx context.Context, obj client.Object) error {
	switch o := obj.(type) {

	case *environments1alpha1.Build:
		_, err := a.build.Resolve(ctx, o)
		return err

	case *environments1alpha1.Deployment:
		_, err := a.deployment.Resolve(ctx, o)
		return err

	case *environments1alpha1.ServiceUnit:
		_, err := a.serviceunit.Resolve(ctx, o)
		return err

	case *eventsv1alpha1.GitHubEvent:
		_, err := a.githubevent.Resolve(ctx, o)
		return err

	case *sourcesv1alpha1.GitRepository:
		_, err := a.gitrepository.Resolve(ctx, o)
		return err

	default:
		return fmt.Errorf(
			"unsupported object type %T in resolution adapter",
			obj,
		)
	}
}
