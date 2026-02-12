package environment

import (
	"context"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

type Adapter struct {
	// future deps:
	// client client.Client
	// log    logr.Logger
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Resolve(
	ctx context.Context,
	environment *environmentv1.Environment,
) (*ResolvedEnvironment, error) {
	return ResolveEnvironment(environment)
}
