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
Package environment implements resolution for the Environment CR.

This file owns the contract adapter — the one-way projection from the resolved
runtime spec (ResolvedEnvironmentSpec) to the protobuf contract type
(environmentscontractv1alpha1.EnvironmentSpec).

The contract type is consumed by infrastructure-layer concerns only:
  - Execution hash computation for environment deduplication
  - Spec comparison across reconciliation cycles
  - Audit pipelines

Controllers and domain logic MUST NOT consume the returned contract value.

Key generated type facts:
  - EnvironmentType → *v1.EnvironmentType wrapper message (Type field carries enum)
  - All CR refs (Build, Deployment, Route, Package, ServiceUnits, BuildTriggers)
    → *ObjectRef / []*ObjectRef — a single local message with only a Name field.
    There are no typed ref messages (EnvironmentBuildRef etc.).
*/
package environment

import (
	commoncontractv1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/common/v1"
	environmentscontractv1alpha1 "github.com/ntlaletsi70/blanketops-environments-contract/blanketops/environments/v1alpha1"
)

// ToEnvironmentContract projects the resolved runtime spec into a protobuf
// environmentscontractv1alpha1.EnvironmentSpec for infrastructure consumers
// (hashing, comparison, audit).
//
// EnvironmentType is projected as a *v1.EnvironmentType wrapper message.
// All CR references (Build, Deployment, Route, Package, ServiceUnits,
// BuildTriggers) are projected as *ObjectRef — the single generic reference
// type used throughout the environment proto.
//
// ⚠️ ONE-WAY adapter. The returned value MUST NOT be fed back into any
// controller or domain logic path.
func (s *ResolvedEnvironmentSpec) ToEnvironmentContract() *environmentscontractv1alpha1.EnvironmentSpec {
	if s == nil {
		return nil
	}

	spec := &environmentscontractv1alpha1.EnvironmentSpec{
		ApplicationName: s.ApplicationName,
		Branch:          s.Branch,
		GitOwner:        s.GitOwner,
		Version:         s.Version,
		Description:     s.Description,
		EnvironmentType: toEnvironmentTypeWrapper(s.EnvironmentType),
	}

	// ------------------------------------------------
	// Build ref (OPTIONAL).
	// ------------------------------------------------
	if s.Build != "" {
		spec.Build = &environmentscontractv1alpha1.ObjectRef{Name: s.Build}
	}

	// ------------------------------------------------
	// BuildTrigger refs (OPTIONAL).
	// ------------------------------------------------
	for _, name := range s.BuildTriggers {
		spec.BuildTriggers = append(spec.BuildTriggers,
			&environmentscontractv1alpha1.ObjectRef{Name: name},
		)
	}

	// ------------------------------------------------
	// ServiceUnit refs (OPTIONAL).
	// ------------------------------------------------
	for _, name := range s.ServiceUnits {
		spec.ServiceUnits = append(spec.ServiceUnits,
			&environmentscontractv1alpha1.ObjectRef{Name: name},
		)
	}

	// ------------------------------------------------
	// Deployment ref (OPTIONAL).
	// ------------------------------------------------
	if s.Deployment != "" {
		spec.Deployment = &environmentscontractv1alpha1.ObjectRef{Name: s.Deployment}
	}

	// ------------------------------------------------
	// Route ref (OPTIONAL).
	// ------------------------------------------------
	if s.Route != "" {
		spec.Route = &environmentscontractv1alpha1.ObjectRef{Name: s.Route}
	}

	// ------------------------------------------------
	// Package ref (OPTIONAL).
	// ------------------------------------------------
	if s.Package != "" {
		spec.Package = &environmentscontractv1alpha1.ObjectRef{Name: s.Package}
	}

	return spec
}

// toEnvironmentTypeWrapper maps an environment type string to the corresponding
// *v1.EnvironmentType wrapper message. Unknown strings map to UNSPECIFIED —
// environment type is user-provided and an unknown value should not fail
// the projection.
func toEnvironmentTypeWrapper(t string) *commoncontractv1.EnvironmentType {
	switch t {
	case "ENVIRONMENT_TYPE_DEVELOPMENT", "development":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_DEVELOPMENT,
		}
	case "ENVIRONMENT_TYPE_STAGING", "staging":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_STAGING,
		}
	case "ENVIRONMENT_TYPE_PRODUCTION", "production":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_PRODUCTION,
		}
	case "ENVIRONMENT_TYPE_TESTING", "testing":
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_TESTING,
		}
	default:
		return &commoncontractv1.EnvironmentType{
			Type: commoncontractv1.EnvironmentType_ENVIRONMENT_TYPE_UNSPECIFIED,
		}
	}
}
