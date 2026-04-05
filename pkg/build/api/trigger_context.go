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
	buildv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/build/domain"
)

// ExtractTriggerContext derives execution trigger metadata from Build annotations.
//
// This function is intentionally Kubernetes-facing only.
// It does NOT touch contract or resolution logic.
func ExtractTriggerContext(build *buildv1.Build) domain.TriggerContext {
	ann := build.Annotations
	if ann == nil {
		return domain.TriggerContext{
			Type: "manual",
		}
	}

	tc := domain.TriggerContext{
		Type: ann["build.blanketops.dev/trigger-type"],
		Ref:  ann["build.blanketops.dev/trigger-ref"],
		SHA:  ann["build.blanketops.dev/trigger-sha"],
	}

	if attempt := ann["build.blanketops.dev/retry-attempt"]; attempt != "" {
		tc.RetryAttempt = attempt
	}

	if tc.Type == "" {
		tc.Type = "manual"
	}

	return tc
}
