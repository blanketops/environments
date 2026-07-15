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
Package packages implements resolution for the Package CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedPackageSpec) to the protobuf contract type
(contractv1.PackageSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for package deduplication
  - Spec comparison across Carvel kapp reconciliation cycles
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.
*/
package packages

import (
	contractv1 "github.com/blanketops/environments-contract/blanketops/environments/v1alpha1"
)

// ToPackageContract projects the resolved runtime Package spec into a
// protobuf contractv1.PackageSpec for infrastructure consumers
// (hashing, comparison, audit).
//
// StateRepository and Maintainers are omitted from the output when nil or
// empty — both are optional in the Package CR and produce no contract fields
// when not configured.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedPackageSpec) ToPackageContract() *contractv1.PackageSpec {
	if s == nil {
		return nil
	}

	out := &contractv1.PackageSpec{
		Enabled:     s.Enabled,
		Name:        s.Name,
		Version:     s.Version,
		Description: s.Description,
		DiffEnabled: s.DiffEnabled,
		Repository: &contractv1.PackageRepository{
			Url:               s.PackageRepository.URL,
			CredentialsSecret: s.PackageRepository.CredentialsSecret,
		},
	}

	// ------------------------------------------------
	// State repository (OPTIONAL).
	//
	// Omitted when nil — not all packages track deployment state via GitOps.
	// ------------------------------------------------
	if s.StateRepository != nil {
		out.StateRepository = &contractv1.StateRepository{
			Url:         s.StateRepository.URL,
			CloneSecret: s.StateRepository.CloneSecret,
			Strategy:    s.StateRepository.Strategy,
			Path:        s.StateRepository.Path,
			// Ref: &contractv1.PackageRef{
			// 	Branch: s.StateRepository.Ref.Branch,
			// 	Tag:    s.StateRepository.Ref.Tag,
			// 	Commit: s.StateRepository.Ref.Commit,
			// },
		}
	}

	// ------------------------------------------------
	// Maintainers (OPTIONAL).
	//
	// Omitted when empty — packages without declared maintainers are valid.
	// ------------------------------------------------
	for _, m := range s.Maintainers {
		out.Maintainers = append(out.Maintainers, &contractv1.Maintainer{
			Name:  m.Name,
			Email: m.Email,
		})
	}

	return out
}
