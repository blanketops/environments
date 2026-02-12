package domain

// Result represents the outcome of attempting to realize
// a GitRepository registration.
type Result struct {
	State  State
	Reason string
}

// Success returns a successful realization result.
func Success() Result {
	return Result{
		State:  StateReady,
		Reason: "repository and webhooks realized",
	}
}

// Pending returns a pending realization result.
func Pending(reason string) Result {
	return Result{
		State:  StatePending,
		Reason: reason,
	}
}

// Failure returns a failed realization result.
func Failure(reason string) Result {
	return Result{
		State:  StateError,
		Reason: reason,
	}
}
