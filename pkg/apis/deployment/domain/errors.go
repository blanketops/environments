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

package domain

import (
	"errors"
	"fmt"
	"strings"
)

// PhaseFailed is the phase of a Deployment, or of a ServiceUnit within it,
// that could not be applied.
const PhaseFailed = "Failed"

// ErrDeploymentFailed reports that executing a Deployment ended with at
// least one ServiceUnit that could not be applied.
var ErrDeploymentFailed = errors.New("deployment failed")

// Failure returns ErrDeploymentFailed, wrapped with what failed, for a result
// whose phase is Failed. It returns nil for any other result: the execution
// did not fail, even if it is not finished.
//
// A ServiceUnit that cannot be applied does not abort the others, so it is
// reported in the result and not as an error from the execution. Failure is
// how a caller turns that result back into an error.
func (r *DeploymentResult) Failure() error {
	if r == nil || r.Phase != DeploymentPhase(PhaseFailed) {
		return nil
	}

	var failed []string
	for _, su := range r.ServiceUnits {
		if su.Phase != ServiceUnitPhase(PhaseFailed) {
			continue
		}
		if su.Error != "" {
			failed = append(failed, fmt.Sprintf("serviceunit %s: %s", su.Name, su.Error))
		} else {
			failed = append(failed, fmt.Sprintf("serviceunit %s failed", su.Name))
		}
	}

	switch {
	case len(failed) > 0:
		return fmt.Errorf("%w: %s", ErrDeploymentFailed, strings.Join(failed, "; "))
	case r.Message != "":
		return fmt.Errorf("%w: %s", ErrDeploymentFailed, r.Message)
	default:
		return ErrDeploymentFailed
	}
}
