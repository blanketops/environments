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

package serviceunit

import "fmt"

func ErrBuildNotReady(name string) error {
	return fmt.Errorf("build for serviceunit %q not ready", name)
}

// ErrInvalidServiceUnit indicates a semantic error in a resolved ServiceUnit.
// This means the resolver violated an invariant or the contract is invalid.
func ErrInvalidServiceUnit(name, reason string) error {
	return fmt.Errorf(
		"invalid serviceunit %q: %s",
		name,
		reason,
	)
}
