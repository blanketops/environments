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
Package esmerge converges the ExternalSecret spec fields this platform owns
without fighting external-secrets' defaulting webhook.

ESO mutates every stored ExternalSecret, adding target.creationPolicy,
target.deletionPolicy, target.template.engineVersion and
target.template.mergePolicy. A reconciler that compares its minimal desired
spec against the stored spec with a whole-object DeepEqual always sees a diff,
replaces the whole spec (stripping the ESO fields), and ESO re-adds them — an
endless strip/re-default flap. When two mediators reconcile the SAME shared
ExternalSecret (e.g. Build and Deployment both own git-ssh-credentials) that
flap also produces resourceVersion churn and Update conflicts that wedge the
slower reconciler. Merging only the managed subset converges instead.
*/
package esmerge

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReconcileManagedSpec copies the managed spec fields (refreshInterval,
// secretStoreRef, data, target.name, target.template.type) from desired onto
// existing, leaving ESO-defaulted fields intact, and reports whether existing
// changed. existing is only mutated when a managed field actually differs, so
// a converged object yields false and no Update is issued.
func ReconcileManagedSpec(existing, desired *unstructured.Unstructured) bool {
	es, _, _ := unstructured.NestedMap(existing.Object, "spec")
	if es == nil {
		es = map[string]any{}
	}
	ds, _, _ := unstructured.NestedMap(desired.Object, "spec")
	if ds == nil {
		ds = map[string]any{}
	}

	changed := false
	for _, k := range []string{"refreshInterval", "secretStoreRef", "data"} {
		if _, ok := ds[k]; ok && !reflect.DeepEqual(es[k], ds[k]) {
			es[k] = ds[k]
			changed = true
		}
	}

	if dt, ok := ds["target"].(map[string]any); ok {
		et, _ := es["target"].(map[string]any)
		if et == nil {
			et = map[string]any{}
		}
		if n, ok := dt["name"]; ok && !reflect.DeepEqual(et["name"], n) {
			et["name"] = n
			changed = true
		}
		if dtt, ok := dt["template"].(map[string]any); ok {
			ett, _ := et["template"].(map[string]any)
			if ett == nil {
				ett = map[string]any{}
			}
			if ty, ok := dtt["type"]; ok && !reflect.DeepEqual(ett["type"], ty) {
				ett["type"] = ty
				changed = true
			}
			et["template"] = ett
		}
		es["target"] = et
	}

	if changed {
		_ = unstructured.SetNestedMap(existing.Object, es, "spec")
	}
	return changed
}
