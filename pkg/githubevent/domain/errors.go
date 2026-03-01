package domain

import "errors"

var (
	// Structural / identity errors
	ErrMissingRepository = errors.New("missing repository")
	ErrInvalidRepository = errors.New("invalid repository identifier")

	// Event semantics
	ErrMissingEventType = errors.New("missing event type")
	ErrUnsupportedEvent = errors.New("unsupported github event type")

	// Git context
	ErrMissingRef    = errors.New("missing git ref")
	ErrMissingCommit = errors.New("missing commit sha")

	// Actor / provenance
	ErrMissingActor = errors.New("missing event actor")

	// Lifecycle / processing
	ErrAlreadyHandled = errors.New("event already handled")
	ErrInvalidState   = errors.New("invalid event state transition")
)
