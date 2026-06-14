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
Package build implements resolution for the Build CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedBuildSpec) to the protobuf contract type (contractv1.BuildSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation (utils.ComputeExecutionHash)
  - Spec comparison across retry attempts
  - Audit and observability pipelines

Controllers and domain logic MUST NOT consume the returned contract value.
The resolved runtime spec (ResolvedBuildSpec) is the authoritative type for
all reconciliation decisions. The contract is a serialisation artifact, not
a source of truth.
*/
package build

import contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"

// ToBuildContract projects the resolved runtime spec into a protobuf
// contractv1.BuildSpec for infrastructure consumers (hashing, comparison,
// audit). Only non-zero fields are populated — nil sub-specs in the resolved
// contract produce nil fields in the output rather than empty structs.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path. Controllers operate on ResolvedBuildSpec,
// never on the contract type.
func (s *ResolvedBuildSpec) ToBuildContract() *contractv1.BuildSpec {
	// Nil guard — no spec, no contract. Callers that pass nil get nil back
	// rather than a panic, keeping hash computation safe at the call site.
	if s == nil {
		return nil
	}

	out := &contractv1.BuildSpec{
		Image: s.Image,
	}

	// ------------------------------------------------
	// Source (Git).
	//
	// Omitted when URL is empty — a Build without a source URL is invalid
	// and will have been caught by resolution before reaching this point.
	// ------------------------------------------------
	if s.Source.URL != "" {
		out.Source = &contractv1.GitSource{
			Url:         s.Source.URL,
			Revision:    s.Source.Revision,
			ContextDir:  s.Source.ContextDir,
			CloneSecret: s.Source.CloneSecret,
		}
	}

	// ------------------------------------------------
	// Strategy.
	//
	// Omitted when Name is empty — strategy is required by resolution
	// so this guard is a safety net, not a valid code path.
	// ------------------------------------------------
	if s.Strategy.Name != "" {
		out.Strategy = &contractv1.BuildStrategy{
			Name: s.Strategy.Name,
		}
		// Kind is a protobuf enum pointer; we leave it nil here to avoid
		// converting the resolved string to the enum pointer in this adapter.
	}

	// ------------------------------------------------
	// Service account.
	// ------------------------------------------------
	if s.ServiceAccount != nil {
		out.ServiceAccount = &contractv1.ServiceAccount{
			Name:   s.ServiceAccount.Name,
			Secret: s.ServiceAccount.Secret,
		}
	}

	// ------------------------------------------------
	// Build policy (retry).
	//
	// Both Policy and Retry are checked for nil independently — a Build
	// may have a Policy without a Retry sub-policy configured.
	// ------------------------------------------------
	if s.Policy != nil && s.Policy.Retry != nil {
		out.Policy = &contractv1.BuildPolicy{
			Retry: &contractv1.RetryPolicy{
				OnFailure:   s.Policy.Retry.OnFailure,
				MaxAttempts: uint32(s.Policy.Retry.MaxAttempts),
			},
		}
	}

	return out
}
