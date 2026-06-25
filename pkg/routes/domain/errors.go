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
This file owns the sentinel errors for the Route domain layer.

These errors are returned by provider implementations when pre-conditions are
not met. The application service wraps them with context before returning to
the controller. Callers may use errors.Is for structured matching.

See also:
  - pkg/routes/api/knative.go — primary consumer of these errors
  - pkg/routes/application/service.go — wraps and surfaces them to the controller
*/
package domain

import "errors"

var (
	// ErrRouteNil is returned when a nil Route aggregate is passed to a provider.
	ErrRouteNil = errors.New("route is nil")

	// ErrHostEmpty is returned when the Route.Host field is empty.
	ErrHostEmpty = errors.New("host is required")

	// ErrRuntimeEmpty is returned when the Route.Runtime field is empty.
	ErrRuntimeEmpty = errors.New("runtime is required")

	// ErrRuntimeUnknown is returned when the Runtime value is not recognised
	// by the backend selector.
	ErrRuntimeUnknown = errors.New("unknown runtime")

	// ErrServiceRefEmpty is returned by the Knative provider when ServiceRef
	// is absent. Without a service reference the DomainMapping cannot be created.
	ErrServiceRefEmpty = errors.New("service reference is required for knative-service runtime")
)
