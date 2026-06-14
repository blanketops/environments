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

Config drives logger construction — which sinks are active, at what level,
and whether structured or human-readable output is used. DefaultConfig
returns a development-friendly baseline (console, info level) suitable for
local runs and CI. Production deployments override via environment or flags.

Supported sinks:
  - Console: structured JSON (production) or human-readable (development).
  - File: appends structured output to FilePath. Off by default.
  - Papertrail: remote syslog sink for centralised log aggregation.
    Requires PapertrailAddr (host:port) and PapertrailTag.
*/
package logging

// Config holds all logging configuration for the controller.
// Passed to the logger constructor at startup — changes after
// construction have no effect.
type Config struct {
	// Development switches the console sink to human-readable output
	// (zap development mode). Set false in production for structured JSON.
	Development bool
	
	// Console enables the console (stdout) sink.
	Console bool
	
	// File enables the file sink. Output is appended to FilePath.
	File     bool
	
	// FilePath is the log file destination when File is true.
	// Defaults to /tmp/blanketops-environment-controller.log.
	FilePath string
	
	// Level controls the minimum log level. One of: "debug", "info",
	// "warn", "error". Case-insensitive. Defaults to "info".
	Level string

	// EnablePapertrail enables the remote Papertrail syslog sink.
	EnablePapertrail bool
	
	// PapertrailAddr is the Papertrail destination in host:port form
	// (e.g. "logs.papertrailapp.com:12345"). Required when EnablePapertrail
	// is true.
	PapertrailAddr string
	
	// PapertrailTag is the program tag attached to all Papertrail log
	// entries. Typically the service name or environment identifier.
	PapertrailTag string
}

// DefaultConfig returns a development-friendly logging configuration.
// Console output is enabled at info level with human-readable formatting.
// File and Papertrail sinks are off. Suitable for local development and CI;
// override individual fields for staging and production deployments.
func DefaultConfig() Config {
	return Config{
		Development: true,
		Console:     true,
		File:        false,
		FilePath:    "/tmp/blanketops-environment-controller.log",
		Level:       "info",
	}
}