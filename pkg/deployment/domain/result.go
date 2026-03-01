/*
Copyright 2025.
*/

package domain

import "time"

//
// ---- Deployment Result ----
//

// Result represents the outcome of a deployment execution
type Result struct {
	// Overall success flag
	Success bool

	// Partial indicates mixed success (some ServiceUnits failed)
	Partial bool

	// Per-ServiceUnit results
	ServiceUnits []ServiceUnitResult

	// Error, if the deployment failed or partially failed
	Error *DeploymentError

	// When this result was produced
	CompletedAt time.Time
}

// ServiceUnitResult represents the result of deploying a single ServiceUnit
// ---- ServiceUnit Result ----
//

//
// ---- Constructors ----
//

// SuccessResult creates a fully successful deployment result
func SuccessResult(units []ServiceUnitResult) Result {
	return Result{
		Success:      true,
		Partial:      false,
		ServiceUnits: units,
		CompletedAt:  time.Now(),
	}
}

// PartialResult creates a partially successful deployment result
func PartialResult(units []ServiceUnitResult, err DeploymentError) Result {
	return Result{
		Success:      false,
		Partial:      true,
		ServiceUnits: units,
		Error:        &err,
		CompletedAt:  time.Now(),
	}
}

// FailureResult creates a failed deployment result
func FailureResult(err DeploymentError) Result {
	return Result{
		Success:     false,
		Partial:     false,
		Error:       &err,
		CompletedAt: time.Now(),
	}
}
