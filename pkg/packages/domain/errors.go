package domain

import "fmt"

// -----------------------------------------------------------------------------
// Base domain error
// -----------------------------------------------------------------------------

type PackageError interface {
	error
	Reason() string
	Retryable() bool
}

// -----------------------------------------------------------------------------
// Invalid specification
// -----------------------------------------------------------------------------

type InvalidSpecError struct {
	Msg string
}

func (e InvalidSpecError) Error() string {
	return fmt.Sprintf("invalid package spec: %s", e.Msg)
}

func (e InvalidSpecError) Reason() string {
	return "InvalidSpec"
}

func (e InvalidSpecError) Retryable() bool {
	return false
}

// -----------------------------------------------------------------------------
// Repository resolution errors
// -----------------------------------------------------------------------------

type RepositoryError struct {
	RepoURL string
	Msg     string
}

func (e RepositoryError) Error() string {
	return fmt.Sprintf("repository error [%s]: %s", e.RepoURL, e.Msg)
}

func (e RepositoryError) Reason() string {
	return "RepositoryError"
}

func (e RepositoryError) Retryable() bool {
	return true
}

// -----------------------------------------------------------------------------
// kapp execution errors
// -----------------------------------------------------------------------------

type KappExecutionError struct {
	Action KappAction
	Output string
	Msg    string
}

func (e KappExecutionError) Error() string {
	return fmt.Sprintf("kapp %s failed: %s", e.Action, e.Msg)
}

func (e KappExecutionError) Reason() string {
	return "KappExecutionFailed"
}

func (e KappExecutionError) Retryable() bool {
	// kapp failures are usually retryable unless spec is broken
	return true
}

// -----------------------------------------------------------------------------
// Diff required but not allowed
// -----------------------------------------------------------------------------

type DiffRequiredError struct {
	Msg string
}

func (e DiffRequiredError) Error() string {
	return fmt.Sprintf("diff required: %s", e.Msg)
}

func (e DiffRequiredError) Reason() string {
	return "DiffRequired"
}

func (e DiffRequiredError) Retryable() bool {
	return false
}

// -----------------------------------------------------------------------------
// State mismatch (drift detected)
// -----------------------------------------------------------------------------

type DriftDetectedError struct {
	Msg string
}

func (e DriftDetectedError) Error() string {
	return fmt.Sprintf("state drift detected: %s", e.Msg)
}

func (e DriftDetectedError) Reason() string {
	return "DriftDetected"
}

func (e DriftDetectedError) Retryable() bool {
	return true
}
