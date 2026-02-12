package intent

import "fmt"

func ErrServiceUnitNotFound(name string) error {
	return fmt.Errorf("serviceunit %q not found", name)
}

func ErrBuildNotReady(name string) error {
	return fmt.Errorf("build for serviceunit %q not ready", name)
}

// ErrInvalidServiceUnit indicates a semantic error in a resolved ServiceUnit.
// This means the resolver violated an invariant or the contract is invalid.
func ErrInvalidServiceUnit(name, reason string) error {
	return fmt.Errorf(
		"invalid serviceunit %q: %s",
		name,
		reason,
	)
}

// ErrInvalidDeployment indicates a semantic error in a resolved Deployment.
// This means the resolver violated an invariant or the contract is invalid.
func ErrInvalidDeployment(reason string) error {
	return fmt.Errorf(
		"invalid deployment: %s",
		reason,
	)
}
