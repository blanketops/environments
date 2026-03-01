package intent

import "time"

type ServiceUnitResult struct {
	Name string

	Phase   ServiceUnitPhase
	Image   string
	Runtime Runtime

	Message string
	Error   string

	LastTransitionTime time.Time
}

type ServiceUnitPhase string

const (
	ServiceUnitPending   ServiceUnitPhase = "Pending"
	ServiceUnitResolving ServiceUnitPhase = "Resolving"
	ServiceUnitDeploying ServiceUnitPhase = "Deploying"
	ServiceUnitReady     ServiceUnitPhase = "Ready"
	ServiceUnitFailed    ServiceUnitPhase = "Failed"
)
