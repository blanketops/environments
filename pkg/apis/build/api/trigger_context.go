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

package api

import (
	buildv1 "github.com/BlanketOps/environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/apis/build/domain"
)

// ExtractTriggerContext derives execution trigger metadata from Build
// annotations. It is the only place in the provider layer that reads
// Kubernetes object metadata — all other inputs arrive via the resolved spec.
//
// Annotations are written by the BuildTrigger domain when it patches the Build
// CR before dispatching. If none are present the trigger is assumed to be
// manual (operator or CI invoking the reconciler directly).
//
// The returned TriggerContext is fed into utils.ComputeExecutionHash to produce
// the execution identity used for BuildRun deduplication.
func ExtractTriggerContext(build *buildv1.Build) domain.TriggerContext {
	ann := build.Annotations
	if ann == nil {
		return domain.TriggerContext{Type: "manual"}
	}

	tc := domain.TriggerContext{
		Type: ann["build.blanketops.dev/trigger-type"],
		Ref:  ann["build.blanketops.dev/trigger-ref"],
		SHA:  ann["build.blanketops.dev/trigger-sha"],
	}

	if attempt := ann["build.blanketops.dev/retry-attempt"]; attempt != "" {
		tc.RetryAttempt = attempt
	}

	// Default to manual when the annotation is absent or empty — ensures
	// the execution hash is always well-formed.
	if tc.Type == "" {
		tc.Type = "manual"
	}

	return tc
}
