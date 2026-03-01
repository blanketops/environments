package domain

// State represents the lifecycle state of a GitRepository registration.
type State string

const (
	// StatePending means the repository has been declared
	// but has not yet been realized.
	StatePending State = "Pending"

	// StateReady means the repository and all required webhooks
	// are realized and match desired intent.
	StateReady State = "Ready"

	// StateError means realization failed and requires attention.
	StateError State = "Error"
)

// IsTerminal indicates whether the state is terminal.
func (s State) IsTerminal() bool {
	return s == StateReady || s == StateError
}
