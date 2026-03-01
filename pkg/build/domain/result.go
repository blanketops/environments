package domain

import (
	shipwrightvbeta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This is what *every* build provider returns.
// Unified result regardless of Buildah, Kaniko, Buildpacks.
type BuildResult struct {
	Success      bool
	Triggered    bool
	Message      string
	Logs         []string
	ExecutionRef string
	BuildHash    string

	ArtifactRef string

	ShipwrightBuild    *shipwrightvbeta1.Build
	ShipwrightBuildRun *shipwrightvbeta1.BuildRun

	// --------------------
	// Retry (AUTHORITATIVE)
	// --------------------

	OnFailure bool

	LastFailureAt *metav1.Time
}
