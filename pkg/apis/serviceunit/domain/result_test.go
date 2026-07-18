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

import "testing"

func TestResult_Ready(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		want  bool
	}{
		{"ready", PhaseReady, true},
		{"pending", PhasePending, false},
		{"deploying", PhaseDeploying, false},
		{"failed", PhaseFailed, false},
		{"zero value", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{Phase: tt.phase}
			if got := r.Ready(); got != tt.want {
				t.Errorf("Ready() = %v, want %v", got, tt.want)
			}
		})
	}
}
