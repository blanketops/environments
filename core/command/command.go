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
Package command owns the command model — the atomic unit of intent that
flows through the CQRS engine. Controllers observe Kubernetes events,
translate them into Commands, and hand them to the Engine. The Engine
routes each Command to the correct Domain by GVK. Domains act on the
Command and never interact with the controller layer directly.

The flow is strictly one-directional:

	controller-runtime event → Command → Engine → Domain → reconciliation
*/
package command

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CommandType identifies the lifecycle operation being requested.
// Values align with the three controller-runtime event types and map
// directly to the CQRS create/update/delete intent model.
type CommandType string

const (
	// CmdCreate is dispatched when a new object is observed for the first time.
	CmdCreate CommandType = "create"

	// CmdUpdate is dispatched when an existing object's spec has changed.
	// The CanUpdate predicate on the Domain determines whether the change
	// is significant enough to trigger reconciliation.
	CmdUpdate CommandType = "update"

	// CmdDelete is dispatched when an object is being removed. Domains use
	// this to perform cleanup — cache invalidation, child resource teardown,
	// or finalizer processing.
	CmdDelete CommandType = "delete"
)

// Command is the atomic unit of intent in the BlanketOps CQRS engine.
// It is constructed by a controller from a controller-runtime reconcile
// request and routed by the Engine to the Domain whose GVK matches.
//
// Old and New are populated only for CmdUpdate — they carry the previous
// and current object state so Domain predicates can diff specs without
// a separate API fetch.
type Command struct {
	// Type is the lifecycle operation being requested.
	Type CommandType

	// GVK identifies which Domain should handle this Command.
	GVK schema.GroupVersionKind

	// Obj is the current object instance. Always populated.
	Obj client.Object

	// Old is the pre-update object state. Populated only for CmdUpdate.
	Old client.Object

	// New is the post-update object state. Populated only for CmdUpdate.
	// Equivalent to Obj on update — both are provided for clarity at call sites.
	New client.Object
}

// Name returns the object's name, or empty string if Obj is nil.
func (c Command) Name() string {
	if c.Obj == nil {
		return ""
	}
	return c.Obj.GetName()
}

// Namespace returns the object's namespace, or empty string if Obj is nil.
func (c Command) Namespace() string {
	if c.Obj == nil {
		return ""
	}
	return c.Obj.GetNamespace()
}

// String returns a concise log-friendly representation of the command,
// suitable for structured log values and error messages.
func (c Command) String() string {
	return fmt.Sprintf("[%s %s/%s %s]", c.Type, c.Namespace(), c.Name(), c.GVK.String())
}

// Clone returns a shallow copy of the Command with Obj deep-copied.
// Old and New are not deep-copied — they are reference-copied. Use Clone
// when dispatching a Command to an async worker or retry queue where the
// caller may mutate Obj after dispatch.
func (c Command) Clone() Command {
	return Command{
		Type: c.Type,
		GVK:  c.GVK,
		Obj:  c.Obj.DeepCopyObject().(client.Object),
		Old:  c.Old,
		New:  c.New,
	}
}
