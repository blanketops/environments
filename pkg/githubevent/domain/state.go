package domain

type State string

const (
	StateNew     State = "New"
	StateHandled State = "Handled"
	StateIgnored State = "Ignored"
	StateError   State = "Error"
)
