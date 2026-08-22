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

package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/blanketops/environments/pkg/apis/deployment/api"
	"github.com/blanketops/environments/pkg/apis/deployment/domain"
	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

// K8SStrategy dispatches a DeploymentIntent's Strategy (Rolling, BlueGreen)
// against the Kubernetes runtime, delegating the actual object apply/
// teardown to api.K8SProvider.
type K8SStrategy struct {
	K8S *api.K8SProvider
	Log logr.Logger
}

// NewK8SStrategy constructs a K8SStrategy wrapping the given K8SProvider.
func NewK8SStrategy(k8s *api.K8SProvider, log logr.Logger) *K8SStrategy {
	return &K8SStrategy{
		K8S: k8s,
		Log: log,
	}
}

// Runtime reports that K8SStrategy handles intent.RuntimeKubernetes.
func (s *K8SStrategy) Runtime() intent.Runtime {
	return intent.RuntimeKubernetes
}

// Teardown delegates to K8S.Teardown, deleting the Kubernetes Deployment
// and Service objects this strategy applied for dIntent.
func (s *K8SStrategy) Teardown(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) error {
	return s.K8S.Teardown(ctx, dIntent)
}

// Supports reports whether K8SStrategy can run the given rollout strategy
// (only Rolling and BlueGreen).
func (s *K8SStrategy) Supports(
	strategy intent.Strategy,
) bool {

	switch strategy {
	case intent.StrategyRolling,
		intent.StrategyBlueGreen:
		return true
	default:
		return false
	}
}

// Execute dispatches dIntent's rollout strategy (Rolling, BlueGreen) to the
// matching apply path, delegating the actual object apply to K8S.
func (s *K8SStrategy) Execute(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	if !s.Supports(dIntent.Strategy) {
		return nil, fmt.Errorf(
			"strategy %s not supported for runtime %s",
			dIntent.Strategy,
			s.Runtime(),
		)
	}

	// Parameter renamed to dIntent (not intent) specifically so these case
	// labels can reference the intent package's constants — a parameter
	// named intent here previously shadowed the package, so both cases
	// resolved to dIntent.Strategy itself (switch X { case X: }), making
	// the first case always match and executeBlueGreen unreachable.
	switch dIntent.Strategy {

	case intent.StrategyRolling:
		return s.executeRolling(ctx, dIntent)

	case intent.StrategyBlueGreen:
		return s.executeBlueGreen(ctx, dIntent)

	default:
		return nil, fmt.Errorf("unknown strategy: %s", dIntent.Strategy)
	}
}

func (s *K8SStrategy) executeRolling(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	results := make([]domain.ServiceUnitResult, 0, len(dIntent.ServiceUnits))

	for _, su := range dIntent.ServiceUnits {
		res, err := s.K8S.ApplyServiceUnit(ctx, dIntent, &su)
		if err != nil {
			results = append(results, domain.ServiceUnitResult{
				Name:               su.Name,
				Phase:              domain.ServiceUnitPhase("Failed"),
				Image:              su.Image,
				Runtime:            domain.Runtime(dIntent.Runtime),
				Error:              err.Error(),
				LastTransitionTime: time.Now(),
			})
			continue
		}
		results = append(results, *res)
	}

	return &domain.DeploymentResult{
		Phase:          deriveDeploymentPhase(results),
		Runtime:        domain.Runtime(dIntent.Runtime),
		Strategy:       domain.Strategy(dIntent.Strategy),
		ServiceUnits:   results,
		LastUpdateTime: time.Now(),
	}, nil
}

// For now BlueGreen reuses rolling behavior.
// Later you can split traffic or manage dual deployments.
func (s *K8SStrategy) executeBlueGreen(
	ctx context.Context,
	dIntent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	s.Log.Info("BlueGreen currently mapped to rolling behavior")

	return s.executeRolling(ctx, dIntent)
}

func deriveDeploymentPhase(
	results []domain.ServiceUnitResult,
) domain.DeploymentPhase {

	if len(results) == 0 {
		return domain.DeploymentPhase("Pending")
	}

	allReady := true

	for _, r := range results {
		switch r.Phase {

		case domain.ServiceUnitPhase("Failed"):
			return domain.DeploymentPhase("Failed")

		case domain.ServiceUnitPhase("Ready"):
			// still possibly all ready

		default:
			allReady = false
		}
	}

	if allReady {
		return domain.DeploymentPhase("Ready")
	}

	return domain.DeploymentPhase("Deploying")
}
