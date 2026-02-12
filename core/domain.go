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
