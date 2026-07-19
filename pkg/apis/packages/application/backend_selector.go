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

package application

import (
	packageapi "github.com/blanketops/environments/pkg/apis/packages/api"
	"github.com/blanketops/environments/pkg/intent/package"
)

// BackendSelector is intentionally boring.
// Packages are ALWAYS executed via kapp.
// No strategy selection is allowed.
type BackendSelector struct {
	Kapp packageapi.Provider
}

// NewBackendSelector constructs a fixed kapp executor.
func NewBackendSelector(
	kapp packageapi.Provider,
) *BackendSelector {
	return &BackendSelector{
		Kapp: kapp,
	}
}

// ForIntent always returns kapp.
// Intent does NOT influence execution backend.
func (b *BackendSelector) ForIntent(
	_ *intent.PackageIntent,
) packageapi.Provider {

	if b.Kapp == nil {
		panic("kapp provider must be configured for PackageService")
	}

	return b.Kapp
}
