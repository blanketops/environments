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
package command

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/ntlaletsi70/blanketops-environments/core"
)

// EngineExecutor is the minimal surface the router needs from the engine.
// core.Engine satisfies this implicitly via its Execute method.
// No interface lives in core — this is the consumer-side contract.
type EngineExecutor interface {
	Execute(ctx context.Context, cmd core.Command) error
}

// Router sits between controllers and the core engine.
// It is the only component that knows about both commands and CQRS events.
type Router struct {
	engine EngineExecutor
	bus    EventBus
	logger logr.Logger
}

// NewRouter constructs a Router. The engine argument is typically *core.Engine.
func NewRouter(engine EngineExecutor, bus EventBus, logger logr.Logger) *Router {
	return &Router{
		engine: engine,
		bus:    bus,
		logger: logger.WithName("command.Router"),
	}
}

// Route is the single entry point for all commands.
//  1. Emits CommandReceived so the observer can write "Pending".
//  2. Applies domain-specific enrichments (GitHub, etc.).
//  3. Delegates to the engine with retry-on-conflict handled here.
//  4. Emits terminal success/failure events.
func (r *Router) Route(ctx context.Context, cmd core.Command) error {
	corrID := correlationID(cmd)

	// 1. Emit receipt immediately — closes the status gap.
	if err := r.emit(ctx, CommandReceived{
		CorrelationID: corrID,
		CommandType:   cmd.Type,
		GVK:           cmd.GVK,
		Name:          cmd.Name(),
		Namespace:     cmd.Namespace(),
		Timestamp:     time.Now(),
	}); err != nil {
		r.logger.Error(err, "failed to emit CommandReceived", "corrID", corrID)
	}

	// 2. GitHub domain enrichment.
	if repo, ok := TriggerOnGitHubDomain(cmd); ok {
		if err := r.emit(ctx, ExternalGitHubTriggerEnqueued{
			CorrelationID: corrID,
			Repository:    repo,
			Timestamp:     time.Now(),
		}); err != nil {
			r.logger.Error(err, "failed to emit ExternalGitHubTriggerEnqueued", "corrID", corrID)
		}
	}

	// 3. Execute with retry-on-conflict.
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		currentCmd := cmd
		if attempt > 1 {
			// PatchOnRetry enrichment: mutate command before retry.
			currentCmd = PatchOnRetry(currentCmd, lastErr, attempt)
		}

		lastErr = r.engine.Execute(ctx, currentCmd)
		if lastErr == nil {
			// Success.
			if err := r.emit(ctx, CommandSucceeded{
				CorrelationID: corrID,
				GVK:           cmd.GVK,
				Name:          cmd.Name(),
				Namespace:     cmd.Namespace(),
				Timestamp:     time.Now(),
			}); err != nil {
				r.logger.Error(err, "failed to emit CommandSucceeded", "corrID", corrID)
			}
			return nil
		}

		if !apierrors.IsConflict(lastErr) {
			// Non-retryable error — fail fast.
			break
		}

		r.logger.Info("conflict detected, will retry",
			"corrID", corrID,
			"attempt", attempt,
			"maxRetries", maxRetries,
		)

		if err := r.emit(ctx, CommandRetrying{
			CorrelationID: corrID,
			CommandType:   cmd.Type,
			GVK:           cmd.GVK,
			Attempt:       attempt,
			Reason:        lastErr.Error(),
			Timestamp:     time.Now(),
		}); err != nil {
			r.logger.Error(err, "failed to emit CommandRetrying", "corrID", corrID)
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt*attempt) * 50 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Terminal failure.
	if err := r.emit(ctx, CommandFailed{
		CorrelationID: corrID,
		GVK:           cmd.GVK,
		Name:          cmd.Name(),
		Namespace:     cmd.Namespace(),
		Error:         lastErr.Error(),
		Timestamp:     time.Now(),
	}); err != nil {
		r.logger.Error(err, "failed to emit CommandFailed", "corrID", corrID)
	}

	return lastErr
}

func (r *Router) emit(ctx context.Context, evt Event) error {
	if r.bus == nil {
		return nil
	}
	return r.bus.Emit(ctx, evt)
}

func correlationID(cmd core.Command) string {
	return fmt.Sprintf("%s/%s/%s", cmd.Namespace(), cmd.Name(), cmd.GVK.String())
}
