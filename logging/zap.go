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

This file owns zap pipeline construction. buildZap assembles a tee core
from the enabled sinks (console, file, Papertrail) and returns a fully
configured *zap.Logger. It is the only place in the package where zap
internals are touched — all other files deal in Config, logr.Logger, or
the returned *zap.Logger handle.

Sink assembly:
  - Console: ConsoleEncoder to stdout. Human-readable in development,
    still structured enough for log scrapers in CI.
  - File: JSONEncoder via lumberjack with 100 MB max size and 28-day
    retention. Rotation is handled by lumberjack — no external logrotate
    required.
  - Papertrail: JSONEncoder via syslog/TCP. Nil cores from buildPapertrailCore
    are skipped — a failed Papertrail connection never prevents startup.

All enabled cores share the same encoder config (ISO8601 timestamps under
the "ts" key) and the same parsed log level.
*/
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// buildZap constructs the zap.Logger from cfg by assembling a tee over all
// enabled sink cores. Called once by Init. Returns an error only if core
// construction fails catastrophically — sink-level failures (e.g. Papertrail
// unreachable) are non-fatal and result in that sink being omitted from
// the tee.
func buildZap(cfg Config) (*zap.Logger, error) {
	// Shared encoder config across all sinks. ISO8601 timestamps under "ts"
	// are readable by humans and parseable by most log aggregators.
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var cores []zapcore.Core

	// ------------------------------------------------
	// Console sink.
	//
	// ConsoleEncoder produces human-readable output suited for local
	// development and CI. JSON would be more consistent with the file
	// sink but less readable at a terminal.
	// ------------------------------------------------
	if cfg.Console {
		cores = append(cores,
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(encCfg),
				zapcore.AddSync(os.Stdout),
				parseLevel(cfg.Level),
			),
		)
	}

	// ------------------------------------------------
	// File sink.
	//
	// lumberjack handles rotation — 100 MB max per file, 28-day retention.
	// No external logrotate configuration required.
	// ------------------------------------------------
	if cfg.File {
		cores = append(cores,
			zapcore.NewCore(
				zapcore.NewJSONEncoder(encCfg),
				zapcore.AddSync(&lumberjack.Logger{
					Filename: cfg.FilePath,
					MaxSize:  100, // MB
					MaxAge:   28,  // days
				}),
				parseLevel(cfg.Level),
			),
		)
	}

	// ------------------------------------------------
	// Papertrail sink.
	//
	// buildPapertrailCore returns nil when disabled or unreachable —
	// nil cores are skipped so a Papertrail outage never blocks startup.
	// ------------------------------------------------
	if pt := buildPapertrailCore(cfg, zapcore.NewJSONEncoder(encCfg)); pt != nil {
		cores = append(cores, pt)
	}

	// Tee fans log entries out to all enabled cores simultaneously.
	// Caller() attaches file/line to every entry; stacktraces fire at
	// Error level and above only to avoid noise at info/debug.
	core := zapcore.NewTee(cores...)
	return zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	), nil
}

// parseLevel maps a Config.Level string to a zapcore.Level. Unrecognised
// values default to InfoLevel — the platform never silently drops logs due
// to a misconfigured level string.
func parseLevel(lvl string) zapcore.Level {
	switch lvl {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
