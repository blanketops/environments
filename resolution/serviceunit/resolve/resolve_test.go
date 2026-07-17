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

package resolve

import (
	"testing"
)

func TestResolveServiceUnit_Stub(t *testing.T) {
	// ResolveServiceUnit is currently a stub (real logic commented out
	// pending the protojson decode path) — this test documents current
	// behavior so a change to the stub is caught rather than silently
	// altering what every caller receives.
	resolved, err := ResolveServiceUnit(nil)
	if resolved != nil || err != nil {
		t.Fatalf("expected (nil, nil) from the stub, got (%+v, %v)", resolved, err)
	}
}
