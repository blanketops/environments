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

import "errors"

var (
	// Structural / identity errors
	ErrMissingRepository = errors.New("missing repository")
	ErrInvalidRepository = errors.New("invalid repository identifier")

	// Event semantics
	ErrMissingEventType = errors.New("missing event type")
	ErrUnsupportedEvent = errors.New("unsupported github event type")

	// Git context
	ErrMissingRef    = errors.New("missing git ref")
	ErrMissingCommit = errors.New("missing commit sha")

	// Actor / provenance
	ErrMissingActor = errors.New("missing event actor")

	// Lifecycle / processing
	ErrAlreadyHandled = errors.New("event already handled")
	ErrInvalidState   = errors.New("invalid event state transition")
)
