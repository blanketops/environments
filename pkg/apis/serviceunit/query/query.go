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

// Package query holds the cross-CR lookups a ServiceUnit needs.
//
// A ServiceUnit of type BUILD does not name an image. It names a Build, and
// runs whatever that Build last pushed. Resolution is pure and cannot read
// another object, so the image is looked up here, after resolution and
// before the resolved ServiceUnit is used.
package query

import (
	"context"
	"encoding/json"
	"fmt"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	builddomain "github.com/blanketops/environments/pkg/apis/build/domain"
	serviceunitresolution "github.com/blanketops/environments/resolution/serviceunit/resolve"
)

// BuildKey is the Build a resolved ServiceUnit takes its image from. The
// Build is looked for in the ServiceUnit's namespace unless the reference
// names another. ok is false for a ServiceUnit that is not of type BUILD.
func BuildKey(su *serviceunitresolution.ResolvedServiceUnit) (key types.NamespacedName, ok bool) {
	if su == nil || su.ServiceUnit == nil || su.Spec == nil {
		return types.NamespacedName{}, false
	}
	if su.Spec.Type != commoncontractv1.ServiceUnitType_SERVICE_UNIT_TYPE_BUILD || su.Spec.BuildRef == nil {
		return types.NamespacedName{}, false
	}
	namespace := su.Spec.BuildRef.Namespace
	if namespace == "" {
		namespace = su.ServiceUnit.Namespace
	}
	return types.NamespacedName{Namespace: namespace, Name: su.Spec.BuildRef.Name}, true
}

// BuildImage returns the image the Build last pushed, pinned to its digest,
// as the Build's status records it. It is empty, with no error, for a Build
// that has not pushed an image yet.
func BuildImage(build *environmentsv1alpha1.Build) (string, error) {
	if build == nil || len(build.Status.Contract.Raw) == 0 {
		return "", nil
	}
	var status builddomain.BuildStatus
	if err := json.Unmarshal(build.Status.Contract.Raw, &status); err != nil {
		return "", fmt.Errorf("decode status of build %s/%s: %w", build.Namespace, build.Name, err)
	}
	return status.Image, nil
}

// InjectBuildImage sets the image of a resolved ServiceUnit of type BUILD to
// the one its Build last pushed. Any other type is left as it is.
//
// The image stays empty, with no error, while the Build has not pushed one:
// the ServiceUnit is then waiting for its Build, which is not a failure. A
// Build that cannot be read is an error.
func InjectBuildImage(ctx context.Context, c client.Reader, su *serviceunitresolution.ResolvedServiceUnit) error {
	key, ok := BuildKey(su)
	if !ok {
		return nil
	}

	build := &environmentsv1alpha1.Build{}
	if err := c.Get(ctx, key, build); err != nil {
		return fmt.Errorf("get build %s for serviceunit %s: %w", key, su.ServiceUnit.Name, err)
	}

	image, err := BuildImage(build)
	if err != nil {
		return err
	}
	su.Spec.Image = image
	return nil
}
