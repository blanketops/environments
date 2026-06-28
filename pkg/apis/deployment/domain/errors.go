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
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package domain

//
// ---- Error Kinds ----
//

// ErrorKind classifies *why* a deployment failed
type ErrorKind string

const (
	// Spec / input errors
	ErrInvalidSpec      ErrorKind = "InvalidSpec"
	ErrMissingReference ErrorKind = "MissingReference"
	ErrUnsupported      ErrorKind = "Unsupported"

	// Resolution errors
	ErrBuildNotReady ErrorKind = "BuildNotReady"
	ErrImageResolve  ErrorKind = "ImageResolutionFailed"

	// Runtime errors
	ErrRuntimeUnavailable ErrorKind = "RuntimeUnavailable"
	ErrDeployFailed       ErrorKind = "DeployFailed"

	// Partial / degraded states
	ErrPartialFailure ErrorKind = "PartialFailure"

	// Internal invariants (bugs)
	ErrInvariantViolation ErrorKind = "InvariantViolation"
)

//
// ---- Domain Error ----
//

// DeploymentError is the canonical domain error type
type DeploymentError struct {
	Kind ErrorKind

	// What failed (Deployment, ServiceUnit, Runtime, etc)
	Subject string

	// Optional specific ServiceUnit name
	ServiceUnit string

	// Human-readable explanation
	Message string

	// Underlying cause (never exposed directly to users)
	Cause error
}

func (e DeploymentError) Error() string {
	if e.ServiceUnit != "" {
		return string(e.Kind) + " [" + e.ServiceUnit + "]: " + e.Message
	}
	return string(e.Kind) + ": " + e.Message
}

func (e DeploymentError) Unwrap() error {
	return e.Cause
}

//
// ---- Constructors (IMPORTANT) ----
//

// InvalidSpec indicates the Deployment spec is malformed
func InvalidSpec(msg string) DeploymentError {
	return DeploymentError{
		Kind:    ErrInvalidSpec,
		Subject: "Deployment",
		Message: msg,
	}
}

// MissingServiceUnit indicates a referenced ServiceUnit does not exist
func MissingServiceUnit(name string) DeploymentError {
	return DeploymentError{
		Kind:        ErrMissingReference,
		Subject:     "ServiceUnit",
		ServiceUnit: name,
		Message:     "referenced ServiceUnit not found",
	}
}

// BuildNotReady indicates the ServiceUnit build has not completed
func BuildNotReady(name string) DeploymentError {
	return DeploymentError{
		Kind:        ErrBuildNotReady,
		Subject:     "Build",
		ServiceUnit: name,
		Message:     "build has not completed successfully",
	}
}

// ImageResolutionFailed indicates the image could not be resolved
func ImageResolutionFailed(name, msg string, cause error) DeploymentError {
	return DeploymentError{
		Kind:        ErrImageResolve,
		Subject:     "Image",
		ServiceUnit: name,
		Message:     msg,
		Cause:       cause,
	}
}

// UnsupportedRuntime indicates the runtime is not supported
func UnsupportedRuntime(runtime string) DeploymentError {
	return DeploymentError{
		Kind:    ErrUnsupported,
		Subject: "Runtime",
		Message: "unsupported runtime: " + runtime,
	}
}

// RuntimeUnavailable indicates infrastructure/runtime failure
func RuntimeUnavailable(runtime string, cause error) DeploymentError {
	return DeploymentError{
		Kind:    ErrRuntimeUnavailable,
		Subject: "Runtime",
		Message: "runtime unavailable: " + runtime,
		Cause:   cause,
	}
}

// ServiceUnitDeployFailed indicates a single ServiceUnit failed
func ServiceUnitDeployFailed(name string, cause error) DeploymentError {
	return DeploymentError{
		Kind:        ErrDeployFailed,
		Subject:     "ServiceUnit",
		ServiceUnit: name,
		Message:     "failed to deploy service unit",
		Cause:       cause,
	}
}

// PartialFailure indicates some ServiceUnits succeeded and others failed
func PartialFailure(msg string) DeploymentError {
	return DeploymentError{
		Kind:    ErrPartialFailure,
		Subject: "Deployment",
		Message: msg,
	}
}

// InvariantViolation indicates a programmer error (BUG)
func InvariantViolation(msg string) DeploymentError {
	return DeploymentError{
		Kind:    ErrInvariantViolation,
		Subject: "System",
		Message: msg,
	}
}
