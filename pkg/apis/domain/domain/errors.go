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
This file owns the sentinel errors for the Domain domain layer.

These errors are returned by provider implementations when pre-conditions are
not met. The application service wraps them with context before returning to
the controller. Callers may use errors.Is for structured matching.

See also:
  - pkg/domain/api/knative.go       — primary consumer of these errors
  - pkg/domain/application/service.go — wraps and surfaces them to the controller
*/
package domain

import "errors"

var (
	// ErrDomainNil is returned when a nil Domain aggregate is passed to a provider.
	ErrDomainNil = errors.New("domain is nil")

	// ErrHostEmpty is returned when the Domain.Host field is empty.
	ErrHostEmpty = errors.New("host is required")

	// ErrRouteRefEmpty is returned when Domain.RouteRef is absent.
	ErrRouteRefEmpty = errors.New("route reference is required")

	// ErrStrategyUnknown is returned when the TLSStrategy value is not
	// recognised by the provider. The application layer validates strategy
	// at resolution time — this is a safety fallback.
	ErrStrategyUnknown = errors.New("unknown TLS strategy")

	// ErrCertProvisioning is returned when a custom strategy cert is not yet
	// ready. This is a non-terminal error — the controller requeues.
	ErrCertProvisioning = errors.New("certificate provisioning in progress")
)
