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

package deployment

import "fmt"

// ErrServiceUnitNotFound indicates a Deployment references a ServiceUnit
// name that has no corresponding resolved ServiceUnit.
func ErrServiceUnitNotFound(name string) error {
	return fmt.Errorf("serviceunit %q not found", name)
}

// ErrInvalidDeployment indicates a semantic error in a resolved Deployment.
// This means the resolver violated an invariant or the contract is invalid.
func ErrInvalidDeployment(reason string) error {
	return fmt.Errorf(
		"invalid deployment: %s",
		reason,
	)
}
