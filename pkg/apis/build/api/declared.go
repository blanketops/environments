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

/*
This file owns the helpers that keep the build providers declarative: a
Shipwright object carries what the Build contract declares and nothing it
does not.

Each helper returns nil for an undeclared value, so the field is left off the
object rather than written with an empty name that Shipwright would then try
to resolve. They are shared by the Buildah, Kaniko and Buildpacks providers.
*/
package api

import (
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

// secretRef returns a reference to the named Secret, or nil when the
// contract declares none.
func secretRef(name string) *corev1.LocalObjectReference {
	if name == "" {
		return nil
	}
	return &corev1.LocalObjectReference{Name: name}
}

// runServiceAccount returns the ServiceAccount a BuildRun executes as, or
// nil when the contract declares none and the namespace default applies.
func runServiceAccount(build *buildResolution.ResolvedBuild) *shipwrightv1alpha1.ServiceAccount {
	if build == nil || build.Spec == nil || build.Spec.ServiceAccount == nil || build.Spec.ServiceAccount.Name == "" {
		return nil
	}
	name := build.Spec.ServiceAccount.Name
	return &shipwrightv1alpha1.ServiceAccount{Name: &name}
}
