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
	"strings"

	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/gitrepository/api"
	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/gitrepository/domain"
)

// BackendSelector selects the appropriate repository provider.
type BackendSelector struct {
	GitHub api.Provider
	// Future:
	// GitLab api.Provider
	// Bitbucket api.Provider
}

func NewBackendSelector(github api.Provider) *BackendSelector {
	return &BackendSelector{
		GitHub: github,
	}
}

// ForRepository selects a provider based on repository spec.
func (b *BackendSelector) ForRepository(
	spec domain.GitRepository,
) api.Provider {

	switch strings.ToLower(string(spec.Provider)) {
	case "github":
		return b.GitHub

	default:
		// safe fallback: GitHub
		return b.GitHub
	}
}
