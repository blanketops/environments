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

/*
This file owns the sentinel errors for the ServiceUnit domain layer.

These errors are returned by ServiceUnitService.Reconcile as a final safety
net against resolver invariant violations that reach it directly (as opposed
to the Mapper, which panics loudly on the same violations — see
pkg/apis/serviceunit/application/mapper.go). Callers may use errors.Is for
structured matching.

See also:
  - pkg/apis/serviceunit/application/service.go — surfaces errors as CR conditions
*/
package domain

import "errors"

var (
	// ErrServiceUnitNil is returned when a nil ServiceUnit aggregate is
	// passed to the application service.
	ErrServiceUnitNil = errors.New("serviceunit is nil")

	// ErrTypeUnknown is returned when the Type value is not recognised.
	ErrTypeUnknown = errors.New("unknown serviceunit type")
)
