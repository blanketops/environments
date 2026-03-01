// pkg/buildtrigger/domain/decision.go
package domain

type Decision struct {
	Accepted bool
	Execute  bool

	// Human-readable explanation
	Message string

	// Optional execution reference (BuildRun name, etc)
	ExecutionRef string
}
