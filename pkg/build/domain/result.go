/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
