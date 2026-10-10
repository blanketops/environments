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
	"errors"
	"fmt"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	commoncontractv1 "github.com/blanketops/environments-contract/blanketops/common/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	builddomain "github.com/blanketops/environments/pkg/apis/build/domain"
	buildresolution "github.com/blanketops/environments/resolution/build/resolve"
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

// ErrBuildInOtherNamespace reports a ServiceUnit that names a Build outside
// its own namespace. The image is pulled with the Build's registry secret,
// and a workload can only use a secret in its own namespace.
var ErrBuildInOtherNamespace = errors.New("build is in another namespace")

// BuildPullSecret returns the name of the registry secret the Build declares
// for its image (serviceAccount.secret). It is empty, with no error, for a
// Build that declares none.
func BuildPullSecret(build *environmentsv1alpha1.Build) (string, error) {
	resolved, err := buildresolution.ResolveBuild(build)
	if err != nil {
		return "", fmt.Errorf("resolve build %s/%s: %w", build.Namespace, build.Name, err)
	}
	if resolved.Spec.ServiceAccount == nil {
		return "", nil
	}
	return resolved.Spec.ServiceAccount.Secret, nil
}

// InjectBuildImage sets, on a resolved ServiceUnit of type BUILD, the image
// its Build last pushed and the registry secret that Build declared for it.
// Any other type is left as it is.
//
// The Build has already set up access to its image, so the ServiceUnit does
// not declare credentials of its own and nothing is copied: the secret is
// referenced by name. That is why the Build must be in the ServiceUnit's
// namespace; one that is not is ErrBuildInOtherNamespace.
//
// The image stays empty, with no error, while the Build has not pushed one:
// the ServiceUnit is then waiting for its Build, which is not a failure. A
// Build that cannot be read is an error.
func InjectBuildImage(ctx context.Context, c client.Reader, su *serviceunitresolution.ResolvedServiceUnit) error {
	key, ok := BuildKey(su)
	if !ok {
		return nil
	}
	if key.Namespace != su.ServiceUnit.Namespace {
		return fmt.Errorf("serviceunit %s/%s names build %s: %w", su.ServiceUnit.Namespace, su.ServiceUnit.Name, key, ErrBuildInOtherNamespace)
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
	if image == "" {
		return nil
	}

	secret, err := BuildPullSecret(build)
	if err != nil {
		return err
	}
	su.Spec.ImagePullSecret = secret
	return nil
}
