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

package core

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Domain defines the contract each domain must implement.
// It operates on core.Command (defined in command.go).
type Domain interface {
	// Handle executes the command synchronously (core.Engine calls this).
	Handle(ctx context.Context, cmd Command) error

	// Predicates (used by core.DomainPredicates)
	CanCreate(obj client.Object) bool
	CanUpdate(oldObj, newObj client.Object) bool
	CanDelete(obj client.Object) bool

	// GVK identifies which CRD type this Domain handles.
	GVK() schema.GroupVersionKind
}
