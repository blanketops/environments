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
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"

	intent "github.com/blanketops/environments/pkg/intent/deployment"
)

// capturingSink records every Info() message so the test can prove which
// branch of K8SProvider.Execute's strategy switch actually ran — with an
// empty ServiceUnits list, executeRolling and executeBlueGreen return
// identical *domain.DeploymentResult values (executeBlueGreen just calls
// executeRolling), so the returned value alone can't distinguish "the
// switch correctly dispatched to executeBlueGreen" from "the switch bug
// silently fell through to executeRolling directly" — only
// executeBlueGreen's own log line proves it was reached.
type capturingSink struct {
	mu       sync.Mutex
	messages []string
}

func (s *capturingSink) Init(logr.RuntimeInfo)          {}
func (s *capturingSink) Enabled(int) bool               { return true }
func (s *capturingSink) WithValues(...any) logr.LogSink { return s }
func (s *capturingSink) WithName(string) logr.LogSink   { return s }
func (s *capturingSink) Error(error, string, ...any)    {}
func (s *capturingSink) Info(_ int, msg string, _ ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

func (s *capturingSink) contains(substr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.messages {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

func TestK8SProvider_Execute_StrategyDispatch(t *testing.T) {
	const blueGreenLog = "BlueGreen currently mapped to rolling behavior"

	tests := []struct {
		name             string
		strategy         intent.Strategy
		wantErr          bool
		wantBlueGreenLog bool
	}{
		{"rolling dispatches to executeRolling", intent.StrategyRolling, false, false},
		{"blueGreen dispatches to executeBlueGreen", intent.StrategyBlueGreen, false, true},
		{"canary is rejected by Supports before the switch", intent.StrategyCanary, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &capturingSink{}
			p := &K8SProvider{Log: logr.New(sink)}

			// Empty ServiceUnits — avoids needing a fake client for
			// Patch/Get, since the point of this test is the strategy
			// switch, not the apply logic.
			di := &intent.DeploymentIntent{
				Name:      "test",
				Namespace: "default",
				Runtime:   intent.RuntimeKubernetes,
				Strategy:  tt.strategy,
			}

			result, err := p.Execute(context.Background(), di)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for strategy %q, got nil (result=%+v)", tt.strategy, result)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute(%q): unexpected error: %v", tt.strategy, err)
			}
			if result == nil {
				t.Fatalf("Execute(%q): result is nil", tt.strategy)
			}

			if got := sink.contains(blueGreenLog); got != tt.wantBlueGreenLog {
				t.Errorf("blueGreen log present = %v, want %v — strategy switch did not dispatch as expected", got, tt.wantBlueGreenLog)
			}
		})
	}
}
