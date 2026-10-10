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
	"testing"
)

func TestDeploymentResult_Failure(t *testing.T) {
	tests := []struct {
		name   string
		result *DeploymentResult
		want   string // empty means no failure
	}{
		{name: "nil result", result: nil},
		{name: "ready", result: &DeploymentResult{Phase: "Ready"}},
		{name: "still deploying", result: &DeploymentResult{Phase: "Deploying"}},
		{
			name: "one unit failed, with its error",
			result: &DeploymentResult{Phase: "Failed", ServiceUnits: []ServiceUnitResult{
				{Name: "api", Phase: "Ready"},
				{Name: "web", Phase: "Failed", Error: "deployments.apps is forbidden"},
			}},
			want: "deployment failed: serviceunit web: deployments.apps is forbidden",
		},
		{
			name: "two units failed, one without an error text",
			result: &DeploymentResult{Phase: "Failed", ServiceUnits: []ServiceUnitResult{
				{Name: "api", Phase: "Failed"},
				{Name: "web", Phase: "Failed", Error: "boom"},
			}},
			want: "deployment failed: serviceunit api failed; serviceunit web: boom",
		},
		{name: "failed with only a message", result: &DeploymentResult{Phase: "Failed", Message: "nothing applied"}, want: "deployment failed: nothing applied"},
		{name: "failed with nothing else said", result: &DeploymentResult{Phase: "Failed"}, want: "deployment failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Failure()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("Failure = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrDeploymentFailed) || err.Error() != tt.want {
				t.Errorf("Failure = %v, want %q wrapping ErrDeploymentFailed", err, tt.want)
			}
		})
	}
}
