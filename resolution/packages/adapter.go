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

package packages

import (
	"context"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
)

type Adapter struct {
	// future deps:
	// client client.Client
	// log    logr.Logger
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Resolve(
	ctx context.Context,
	packageResource *environmentv1.Package,
) (*ResolvedPackage, error) {
	return ResolvePackage(packageResource)
}
