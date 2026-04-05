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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CommandType defines the type of action being performed on a Kubernetes object.
// It aligns with CQRS concepts: Create, Update, Delete.
type CommandType string

const (
	CmdCreate CommandType = "create"
	CmdUpdate CommandType = "update"
	CmdDelete CommandType = "delete"
)

// Command represents a single domain event or intention emitted by a controller.
// It is the atomic instruction that the Engine routes to the correct Domain.
type Command struct {
	Type CommandType             // what happened
	GVK  schema.GroupVersionKind // object kind
	Obj  client.Object           // current object instance
	Old  client.Object           // optional: previous object (on update)
	New  client.Object           // optional: new object (on update)
}

// Name returns the object's name if available.
func (c Command) Name() string {
	if c.Obj == nil {
		return ""
	}
	return c.Obj.GetName()
}

// Namespace returns the object's namespace if available.
func (c Command) Namespace() string {
	if c.Obj == nil {
		return ""
	}
	return c.Obj.GetNamespace()
}

// String returns a concise log-friendly representation.
func (c Command) String() string {
	return fmt.Sprintf("[%s %s/%s %s]", c.Type, c.Namespace(), c.Name(), c.GVK.String())
}

// Clone shallow-copies the command (useful for async execution or retries).
func (c Command) Clone() Command {
	return Command{
		Type: c.Type,
		GVK:  c.GVK,
		Obj:  c.Obj.DeepCopyObject().(client.Object),
		Old:  c.Old,
		New:  c.New,
	}
}
