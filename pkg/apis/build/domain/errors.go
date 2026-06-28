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
This file owns the sentinel errors for the build domain.

Sentinel errors allow callers to distinguish known build failures from
unexpected infrastructure errors using errors.Is without importing
implementation details from the provider layer.
*/
package domain

import "errors"

// ErrBuildFailed signals that a build execution completed with a failure
// outcome. Use errors.Is(err, ErrBuildFailed) to match this error.
var ErrBuildFailed = errors.New("build failed")
