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
Package core defines the foundational contracts and primitives shared across
all BlanketOps Environments domains.

This file owns the Engine — the central command router of the CQRS architecture.
The Engine receives Commands from controllers, resolves the correct Domain from
the Registry by GVK, and dispatches Handle. It is the only component that
bridges the controller layer and the domain layer.

The Engine supports two execution modes:

 1. Synchronous (default) — Execute() dispatches directly to the Domain and
    returns the error to the caller (controller-runtime requeue path). This is
    the mode used in all current deployments.

 2. Async worker pool (optional) — Queue() enqueues Commands into a buffered
    channel; a configurable pool of goroutines drains it via workerLoop. This
    mode is opt-in via SetWorkers() and StartWorkers(), intended for high-
    throughput production deployments where controller goroutines should not
    block on Domain execution.

When no workers are configured (workers == 0), Queue() falls through to
Execute() so callers need not distinguish between modes.
*/
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// Engine routes Commands to the correct Domain and supports both synchronous
// and async execution modes. A single Engine instance is shared across all
// controllers registered with the manager.
type Engine struct {
	// registry is the source of Domain lookups by GVK.
	registry *Registry

	// mu guards workers and related fields.
	mu sync.RWMutex

	// queue is the buffered command channel drained by the worker pool.
	// Capacity 1024 provides backpressure headroom before Queue blocks.
	queue chan Command

	// workers is the number of async goroutines draining the queue.
	// Zero means synchronous execution (inline via Execute).
	workers int

	// workerStop signals all workerLoop goroutines to exit cleanly.
	workerStop chan struct{}

	// workerGroup tracks active workers for graceful shutdown.
	workerGroup sync.WaitGroup
	logger      logr.Logger
}

// NewEngine constructs an Engine bound to the given Registry and logger.
// The engine starts in synchronous mode (workers == 0). Call SetWorkers
// and StartWorkers to enable async execution.
func NewEngine(registry *Registry, logger logr.Logger) *Engine {
	return &Engine{
		registry:   registry,
		queue:      make(chan Command, 1024),
		workers:    0,
		workerStop: make(chan struct{}),
		logger:     logger.WithName("core.Engine"),
	}
}

// SetWorkers configures the async worker pool size. Must be called before
// StartWorkers. Zero (default) keeps the Engine in synchronous mode.
func (e *Engine) SetWorkers(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workers = n
}

// Execute dispatches a Command synchronously to the Domain registered for
// cmd.GVK. Returns an error if no Domain is registered or if Handle fails.
// The error propagates back to controller-runtime as a requeue signal.
func (e *Engine) Execute(ctx context.Context, cmd Command) error {
	d, ok := e.registry.GetDomain(cmd.GVK)
	if !ok {
		return fmt.Errorf("no domain registered for GVK %s", cmd.GVK.String())
	}
	e.logger.Info("executing command", "gvk", cmd.GVK.String(), "cmd", cmd.Type, "name", cmd.Name())
	return d.Handle(ctx, cmd)
}

// Queue enqueues a Command for async processing when workers > 0, or falls
// through to Execute when in synchronous mode. Blocks if the queue is full
// and returns ctx.Err() if the context is cancelled while waiting.
func (e *Engine) Queue(ctx context.Context, cmd Command) error {
	if e.workers <= 0 {
		return e.Execute(ctx, cmd)
	}
	select {
	case e.queue <- cmd:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StartWorkers spawns the configured number of async worker goroutines.
// No-op when workers == 0. Safe to call once at manager startup after
// SetWorkers.
func (e *Engine) StartWorkers() {
	e.mu.RLock()
	workers := e.workers
	e.mu.RUnlock()

	if workers <= 0 {
		return
	}

	e.logger.Info("starting engine workers", "workers", workers)
	for i := 0; i < workers; i++ {
		e.workerGroup.Add(1)
		go func(id int) {
			defer e.workerGroup.Done()
			e.workerLoop(id)
		}(i)
	}
}

// StopWorkers signals all worker goroutines to exit and waits for them to
// drain, up to the given timeout. Called during manager shutdown. No-op
// when workers == 0.
func (e *Engine) StopWorkers(timeout time.Duration) {
	if e.workers <= 0 {
		return
	}

	close(e.workerStop)

	done := make(chan struct{})
	go func() {
		e.workerGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		e.logger.Info("timeout waiting for engine workers to stop")
	}
}

// workerLoop is the per-goroutine drain loop. It processes Commands from the
// queue until workerStop is closed. Failed executions are logged — retry,
// dead-letter, and backoff logic can be added here when needed.
func (e *Engine) workerLoop(id int) {
	logger := e.logger.WithValues("worker", id)
	logger.Info("worker started")

	for {
		select {
		case <-e.workerStop:
			logger.Info("worker stopping")
			return
		case cmd := <-e.queue:
			logger.Info("processing command",
				"gvk", cmd.GVK.String(),
				"type", cmd.Type,
				"name", cmd.Name(),
			)
			if err := e.Execute(context.Background(), cmd); err != nil {
				logger.Error(err, "worker execute failed", "cmd", cmd)
			}
		}
	}
}
