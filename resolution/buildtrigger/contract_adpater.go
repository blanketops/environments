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

package buildtrigger

import (
	"fmt"

	contractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/buildtrigger/domain"
)

// ToBuildTriggerContract converts a resolved runtime trigger into a CONTRACT spec
// for hashing, deduplication, comparison, etc.
//
// ⚠️ ONE-WAY ONLY.
// Controllers must NEVER consume the returned value.
func (s *ResolvedBuildTriggerSpec) ToBuildTriggerContract() (*contractv1.BuildTriggerSpec, error) {
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

	out := &contractv1.BuildTriggerSpec{
		Source:     sourceEnum,
		Type:       typeEnum,
		Repository: s.Repository, // already "owner/name"
		Ref:        s.Ref,
		Build: &contractv1.BuildTriggerBuildRef{
			Name: s.Target.Name, // buildRef
		},
	}

	return out, nil
}

// -----------------------------------------------------------------------------
// Enum mapping (domain → contract)
// -----------------------------------------------------------------------------

func mapSource(v domain.TriggerSource) (contractv1.BuildTriggerSource, error) {
	switch v {
	case domain.TriggerSourceGitHub:
		return contractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_GITHUB, nil
	case domain.TriggerSourceGitLab:
		return contractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_GITLAB, nil
	case domain.TriggerSourceManual:
		return contractv1.BuildTriggerSource_BUILD_TRIGGER_SOURCE_MANUAL, nil
	default:
		return 0, fmt.Errorf("unsupported buildtrigger source %q", v)
	}
}

func mapTriggerType(v domain.TriggerType) (contractv1.BuildTriggerType, error) {
	switch v {
	case domain.TriggerTypePush:
		return contractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_PUSH, nil
	case domain.TriggerTypePullRequest:
		return contractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_PULL_REQUEST, nil
	case domain.TriggerTypeManual:
		return contractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_MANUAL, nil
	// case domain.TriggerTypeSchedule:
	// 	return contractv1.BuildTriggerType_BUILD_TRIGGER_TYPE_SCHEDULE, nil
	default:
		return 0, fmt.Errorf("unsupported buildtrigger type %q", v)
	}
}
