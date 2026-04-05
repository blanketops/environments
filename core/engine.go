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
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// Engine routes commands to the appropriate domain.
// Supports both synchronous execution and optional async queueing.
type Engine struct {
	registry *Registry
	mu       sync.RWMutex

	queue       chan Command
	workers     int
	workerStop  chan struct{}
	workerGroup sync.WaitGroup

	logger logr.Logger
}

// NewEngine creates a new engine bound to a Registry and logger.
// Optionally configure async workers later via SetWorkers().
func NewEngine(registry *Registry, logger logr.Logger) *Engine {
	return &Engine{
		registry:   registry,
		queue:      make(chan Command, 1024),
		workers:    0, // default inline
		workerStop: make(chan struct{}),
		logger:     logger.WithName("core.Engine"),
	}
}

// SetWorkers configures worker pool size (optional).
func (e *Engine) SetWorkers(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workers = n
}

// Execute runs a command synchronously against the domain in the registry.
func (e *Engine) Execute(ctx context.Context, cmd Command) error {
	d, ok := e.registry.GetDomain(cmd.GVK)
	if !ok {
		return fmt.Errorf("no domain registered for GVK %s", cmd.GVK.String())
	}

	e.logger.Info("executing command", "gvk", cmd.GVK.String(), "cmd", cmd.Type, "name", cmd.Name())
	return d.Handle(ctx, cmd)
}

// Queue enqueues a command for async processing (or executes inline if no workers).
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

// StartWorkers spawns async workers if workers > 0.
func (e *Engine) StartWorkers() {
	e.mu.RLock()
	workers := e.workers
	e.mu.RUnlock()

	if workers <= 0 {
		return
	}
	e.logger.Info("starting engine workers", "workers", workers)
	for i := 0; i < workers; i++ {
		e.workerGroup.Add(1) // add more workers when deployed to prod
		go func(id int) {
			defer e.workerGroup.Done()
			e.workerLoop(id)
		}(i)
	}
}

// StopWorkers gracefully shuts down the worker pool.
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

// workerLoop handles queued commands.
func (e *Engine) workerLoop(id int) {
	logger := e.logger.WithValues("worker", id)
	logger.Info("worker started")
	for {
		select {
		case <-e.workerStop:
			logger.Info("worker stopping")
			return
		case cmd := <-e.queue:
			logger.Info("processing command", "gvk", cmd.GVK.String(), "type", cmd.Type, "name", cmd.Name())
			if err := e.Execute(context.Background(), cmd); err != nil {
				logger.Error(err, "worker execute failed", "cmd", cmd)
				// Optional: retry, dead-letter, backoff logic here
			}
		}
	}
}
