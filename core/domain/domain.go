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
Package domain owns the Domain interface — the contract every resource
domain must satisfy to participate in the CQRS engine. A Domain is the
single authoritative handler for one CRD kind. It owns the reconciliation
pipeline for that kind from command receipt through to status write.

Registering a Domain with the Engine is the only wiring required — the Engine
handles routing, predicate evaluation, and dispatch. Domains never call each
other directly.
*/
package domain

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	command "github.com/blanketops/environments/core/command"
)

// Domain is the contract every resource domain must implement to participate
// in the BlanketOps CQRS engine.
//
// Each Domain handles exactly one CRD kind, identified by GVK. The Engine
// routes Commands to the matching Domain, evaluates the Domain's predicates
// to determine whether reconciliation should proceed, and calls Handle with
// the resulting Command.
//
// Implementations live under pkg/<domain>/ and follow the pattern:
//
//	type BuildDomain struct { ... }
//	func (d *BuildDomain) GVK() schema.GroupVersionKind { ... }
//	func (d *BuildDomain) Handle(ctx, cmd) error { ... }
//	func (d *BuildDomain) CanCreate(obj) bool { ... }
//	func (d *BuildDomain) CanUpdate(old, new) bool { ... }
//	func (d *BuildDomain) CanDelete(obj) bool { ... }
type Domain interface {
	// GVK identifies the CRD kind this Domain handles. The Engine uses this
	// to route Commands — only one Domain may be registered per GVK.
	GVK() schema.GroupVersionKind

	// Handle executes the reconciliation pipeline for the given Command.
	// Called synchronously by the Engine after predicate evaluation passes.
	// Returning an error signals the Engine to requeue.
	Handle(ctx context.Context, cmd command.Command) error

	// CanCreate reports whether the object should trigger a create pipeline.
	// Implementations typically assert the object's concrete type.
	CanCreate(obj client.Object) bool

	// CanUpdate reports whether the transition from oldObj to newObj should
	// trigger an update pipeline. Implementations typically diff specs to
	// suppress reconciliation on status-only or metadata-only changes.
	CanUpdate(oldObj, newObj client.Object) bool

	// CanDelete reports whether the object should trigger a delete pipeline.
	// Implementations typically assert the object's concrete type.
	CanDelete(obj client.Object) bool
}
