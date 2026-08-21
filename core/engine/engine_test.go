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

package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	command "github.com/blanketops/environments/core/command"
	registry "github.com/blanketops/environments/core/registry"
)

var buildGVK = schema.GroupVersionKind{Group: "environments.blanketops.dev", Version: "v1alpha1", Kind: "Build"}

// fakeDomain records every Command it was asked to Handle. handleFunc, if
// set, overrides the return value and lets a test observe async dispatch.
type fakeDomain struct {
	gvk schema.GroupVersionKind

	mu         sync.Mutex
	handled    []command.Command
	handleErr  error
	handleFunc func(ctx context.Context, cmd command.Command) error
}

func (f *fakeDomain) GVK() schema.GroupVersionKind { return f.gvk }

func (f *fakeDomain) Handle(ctx context.Context, cmd command.Command) error {
	f.mu.Lock()
	f.handled = append(f.handled, cmd)
	f.mu.Unlock()
	if f.handleFunc != nil {
		return f.handleFunc(ctx, cmd)
	}
	return f.handleErr
}

func (f *fakeDomain) CanCreate(client.Object) bool                { return true }
func (f *fakeDomain) CanUpdate(client.Object, client.Object) bool { return true }
func (f *fakeDomain) CanDelete(client.Object) bool                { return true }

func (f *fakeDomain) handledCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.handled)
}

func TestEngine_Execute_DispatchesToRegisteredDomain(t *testing.T) {
	reg := registry.NewRegistry()
	d := &fakeDomain{gvk: buildGVK}
	reg.RegisterDomain(buildGVK, d)

	e := NewEngine(reg, logr.Discard())
	cmd := command.Command{Type: command.CmdCreate, GVK: buildGVK}

	if err := e.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if d.handledCount() != 1 {
		t.Fatalf("expected domain to handle exactly 1 command, got %d", d.handledCount())
	}
}

func TestEngine_Execute_NoDomainRegistered(t *testing.T) {
	reg := registry.NewRegistry()
	e := NewEngine(reg, logr.Discard())

	err := e.Execute(context.Background(), command.Command{GVK: buildGVK})
	if err == nil {
		t.Fatal("expected error when no domain is registered for the GVK")
	}
}

func TestEngine_Execute_PropagatesHandleError(t *testing.T) {
	reg := registry.NewRegistry()
	wantErr := errors.New("handle failed")
	d := &fakeDomain{gvk: buildGVK, handleErr: wantErr}
	reg.RegisterDomain(buildGVK, d)

	e := NewEngine(reg, logr.Discard())
	err := e.Execute(context.Background(), command.Command{GVK: buildGVK})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected Execute to propagate Handle's error, got %v", err)
	}
}

func TestEngine_Queue_SynchronousFallbackWhenNoWorkers(t *testing.T) {
	reg := registry.NewRegistry()
	d := &fakeDomain{gvk: buildGVK}
	reg.RegisterDomain(buildGVK, d)

	e := NewEngine(reg, logr.Discard())
	if err := e.Queue(context.Background(), command.Command{GVK: buildGVK}); err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if d.handledCount() != 1 {
		t.Fatalf("expected synchronous fallback to dispatch immediately, got %d handled", d.handledCount())
	}
}

func TestEngine_Queue_ContextCancelledWhenQueueFull(t *testing.T) {
	reg := registry.NewRegistry()
	d := &fakeDomain{gvk: buildGVK}
	reg.RegisterDomain(buildGVK, d)

	e := NewEngine(reg, logr.Discard())
	e.SetWorkers(1) // switch Queue into the async (channel) path — StartWorkers is never called, so nothing drains it.

	// Fill the queue to its capacity (1024) so the next send blocks.
	for i := 0; i < 1024; i++ {
		if err := e.Queue(context.Background(), command.Command{GVK: buildGVK}); err != nil {
			t.Fatalf("filling queue: unexpected error at %d: %v", i, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := e.Queue(ctx, command.Command{GVK: buildGVK})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled when queue is full and context is done, got %v", err)
	}
}

func TestEngine_StartStopWorkers_ProcessesQueuedCommands(t *testing.T) {
	reg := registry.NewRegistry()
	done := make(chan struct{}, 1)
	d := &fakeDomain{
		gvk: buildGVK,
		handleFunc: func(ctx context.Context, cmd command.Command) error {
			done <- struct{}{}
			return nil
		},
	}
	reg.RegisterDomain(buildGVK, d)

	e := NewEngine(reg, logr.Discard())
	e.SetWorkers(2)
	e.StartWorkers()
	defer e.StopWorkers(time.Second)

	if err := e.Queue(context.Background(), command.Command{GVK: buildGVK}); err != nil {
		t.Fatalf("Queue: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async worker to process the queued command")
	}
}

func TestEngine_StopWorkers_SafeToCallTwice(t *testing.T) {
	reg := registry.NewRegistry()
	e := NewEngine(reg, logr.Discard())
	e.SetWorkers(1)
	e.StartWorkers()

	e.StopWorkers(time.Second)
	e.StopWorkers(time.Second) // must not panic with "close of closed channel"
}

func TestEngine_StartWorkers_NoopWhenZeroWorkers(t *testing.T) {
	reg := registry.NewRegistry()
	e := NewEngine(reg, logr.Discard())
	e.StartWorkers() // should be a no-op
	e.StopWorkers(0) // should also be a no-op, must not block or panic
}
