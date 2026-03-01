package buildtrigger

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
	trigger *environmentv1.BuildTrigger,
) (*ResolvedBuildTrigger, error) {
	return ResolveBuildTrigger(trigger)
}
