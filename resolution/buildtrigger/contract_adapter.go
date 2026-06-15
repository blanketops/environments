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
Package buildtrigger implements resolution for the BuildTrigger CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedBuildTriggerSpec) to the protobuf contract type
(environmentscontractv1alpha1.BuildTriggerSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for deduplication
  - Trigger comparison across webhook deliveries
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.
The resolved runtime spec is the authoritative type for all reconciliation
decisions.

Enum mapping is explicit and fails fast on unknown values — an unrecognised
domain enum reaching this layer indicates a domain/contract version skew
that must not be silently swallowed.

The common proto wrapper pattern: enum values are nested inside wrapper
messages (e.g. BuildTriggerSource_BuildTriggerSource nested inside
BuildTriggerSource). Fields typed as these wrappers must be constructed
as message instances with the enum value set on the value field.
*/
package buildtrigger

import (
	"fmt"

	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
	environmentscontractv1alpha1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

// ToBuildTriggerContract projects the resolved runtime trigger spec into a
// protobuf environmentscontractv1alpha1.BuildTriggerSpec for infrastructure
// consumers (hashing, deduplication, comparison, audit).
//
// Source and Type are projected as wrapper message instances — the common
// proto pattern wraps each enum in a message that carries the value on a
// named field (source/type respectively).
//
// Build is projected as *environmentscontractv1alpha1.BuildRef — the local
// message defined in the environments v1alpha1 package, not a common type.
//
// Returns an error when the domain Source or Type cannot be mapped to a
// contract enum — this indicates a domain/contract version skew and must
// be surfaced, not swallowed.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedBuildTriggerSpec) ToBuildTriggerContract() (*environmentscontractv1alpha1.BuildTriggerSpec, error) {
	if s == nil {
		return nil, nil
	}

	sourceEnum, err := mapSource(s.Source)
	if err != nil {
		return nil, err
	}

	typeEnum, err := mapTriggerType(s.Type)
	if err != nil {
		return nil, err
	}

	out := &environmentscontractv1alpha1.BuildTriggerSpec{
		Source: &commoncontractv1.BuildTriggerSource{
			Source: sourceEnum,
		},
		Type: &commoncontractv1.BuildTriggerType{
			Type: typeEnum,
		},
		Repository: s.Repository, // already "owner/name" — no transformation needed
		Ref:        s.Ref,
		// BuildRef is defined in the environments v1alpha1 package — not a
		// common type. Resolved in the same namespace as the BuildTrigger CR.
		Build: &environmentscontractv1alpha1.BuildRef{
			Name: s.Target.Name,
		},
	}

	return out, nil
}

// -----------------------------------------------------------------------------
// Enum mapping: domain → contract
//
// Each mapper returns the nested enum value type. Unknown values return an
// error rather than a zero value — a zero enum silently maps to the wrong
// proto constant and corrupts hashes and audit records downstream.
// -----------------------------------------------------------------------------

// mapSource maps a domain TriggerSource to the corresponding
// BuildTriggerSource_BuildTriggerSource enum value.
func mapSource(v domain.TriggerSource) (commoncontractv1.BuildTriggerSource_BuildTriggerSource, error) {
	switch v {
	case domain.TriggerSourceGitHub:
		return commoncontractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_GITHUB, nil
	case domain.TriggerSourceGitLab:
		return commoncontractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_GITLAB, nil
	case domain.TriggerSourceManual:
		return commoncontractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_MANUAL, nil
	default:
		return 0, fmt.Errorf("unsupported buildtrigger source %q", v)
	}
}

// mapTriggerType maps a domain TriggerType to the corresponding
// BuildTriggerType_BuildTriggerType enum value.
//
// TriggerTypeSchedule is commented out pending contract support — uncomment
// when BUILD_TRIGGER_TYPE_SCHEDULE is added to the proto.
func mapTriggerType(v domain.TriggerType) (commoncontractv1.BuildTriggerType_BuildTriggerType, error) {
	switch v {
	case domain.TriggerTypePush:
		return commoncontractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_COMMIT, nil
	case domain.TriggerTypePullRequest:
		return commoncontractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_PULL_REQUEST, nil
	case domain.TriggerTypeManual:
		return commoncontractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_MANUAL, nil
	// case domain.TriggerTypeSchedule:
	// 	return commoncontractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_SCHEDULE, nil
	default:
		return 0, fmt.Errorf("unsupported buildtrigger type %q", v)
	}
}
