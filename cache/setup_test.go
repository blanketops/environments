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

package cache

import (
	"testing"

	corecache "github.com/blanketops/environments/core/cache"
)

func TestNewExternal(t *testing.T) {
	tests := []struct {
		name    string
		backend Backend
		wantErr bool
	}{
		{"none", BackendNone, false},
		{"empty defaults to none", "", false},
		// TODO(cache/setup.go): redis and memcached are not wired to real
		// backends yet (both return NoopExternalCache — see the TODOs in
		// setup.go) — asserting Noop here documents current behavior, not
		// the intended end state. Update this test when they're wired.
		{"redis (currently noop)", BackendRedis, false},
		{"memcached (currently noop)", BackendMemcached, false},
		{"unknown backend errors", Backend("bogus"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewExternal(Options{Backend: tt.backend})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for backend %q, got nil", tt.backend)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewExternal(%q): %v", tt.backend, err)
			}
			if _, ok := got.(corecache.NoopExternalCache); !ok {
				t.Errorf("NewExternal(%q) = %T, want NoopExternalCache", tt.backend, got)
			}
		})
	}
}
