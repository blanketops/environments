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
Package logging provides logging configuration and setup for the
BlanketOps Environments controller.

This file owns logger initialisation. The package exposes two logger handles:

  - logr.Logger (rootLog): the controller-runtime compatible interface used
    throughout the platform. Passed to the Engine, domains, and reconcilers.
  - *zap.Logger (rootZap): the underlying zap instance, available for
    components that require direct zap access (e.g. Papertrail sink wiring).

Both are initialised once via sync.Once — repeated calls to Init return the
same loggers. This guarantees a single logging pipeline regardless of how
many times Init is called during startup.
*/
package logging

import (
	"sync"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

var (
	// rootZap is the underlying zap logger instance. Constructed once by Init.
	rootZap *zap.Logger
	
	// rootLog is the logr adapter over rootZap. The primary logger handle
	// passed to controller-runtime and all platform components.
	rootLog logr.Logger
	
	// once ensures Init builds the logging pipeline exactly once regardless
	// of how many callers invoke it during startup.
	once sync.Once
)

// Init constructs the logging pipeline from cfg and returns both logger
// handles. Safe to call multiple times — only the first call builds the
// pipeline; subsequent calls return the already-initialised loggers.
//
// Returns an error if zap construction fails (e.g. invalid log level,
// unreachable Papertrail address). On error, both returned loggers are
// zero-value and the process should not continue.
//
// Typical startup usage:
//
//	log, zapLog, err := logging.Init(logging.DefaultConfig())
//	if err != nil {
//	    os.Exit(1)
//	}
//	ctrl.SetLogger(log)
func Init(cfg Config) (logr.Logger, *zap.Logger, error) {
	var err error
	once.Do(func() {
		rootZap, err = buildZap(cfg)
		if err != nil {
			return
		}
		rootLog = AsLogr(rootZap)
	})
	return rootLog, rootZap, err
}