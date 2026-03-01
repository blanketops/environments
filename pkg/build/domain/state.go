package domain

// IsTerminal reports whether the build execution has reached a terminal state.
//
// A build is terminal once an execution has been triggered and the controller
// has recorded an outcome (success or failure).
//
// Progress and intermediate states are tracked by observers (e.g. BuildRun),
// not by the domain model.
func (s BuildStatus) IsTerminal() bool {
	if !s.Triggered {
		return false
	}

	// Once triggered, the controller records exactly one outcome.
	// Success=true  -> terminal (succeeded)
	// Success=false -> terminal (failed)
	return true
}
