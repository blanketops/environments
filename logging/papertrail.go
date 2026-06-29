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

This file owns Papertrail integration. Two distinct transport paths are
provided:

 1. Syslog (TCP) — buildPapertrailCore wires a zapcore.Core that sends
    structured log entries to Papertrail over syslog/TCP. Integrated into
    the zap pipeline via buildZap. Failure to connect is NON-FATAL — the
    core returns nil and the pipeline continues without the Papertrail sink.

 2. HTTP JSON ingest — SetupPapertrailJSONIngest returns a standalone send
    function for the SolarWinds HTTP ingestion API. Intentionally decoupled
    from zap: no global state, no side effects, safe to call from any
    goroutine. Use this path when you need to ship a specific log line or
    event payload directly to Papertrail outside the normal log pipeline.
*/
package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/syslog"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap/zapcore"
)

// buildPapertrailCore constructs a zapcore.Core that writes to Papertrail
// over syslog/TCP. Returns nil — not an error — when Papertrail is disabled,
// misconfigured, or unreachable. The caller (buildZap) skips nil cores when
// assembling the tee, so a failed Papertrail connection never blocks startup.
func buildPapertrailCore(cfg Config, enc zapcore.Encoder) zapcore.Core {
	if !cfg.EnablePapertrail {
		return nil
	}

	if cfg.PapertrailAddr == "" {
		fmt.Fprintln(os.Stderr, "papertrail enabled but PapertrailAddr is empty")
		return nil
	}

	tag := cfg.PapertrailTag
	if tag == "" {
		tag = "blanketops-environment-operator"
	}

	writer, err := syslog.Dial(
		"tcp",
		cfg.PapertrailAddr,
		syslog.LOG_INFO|syslog.LOG_LOCAL0,
		tag,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Papertrail unavailable: %v\n", err)
		return nil
	}

	return zapcore.NewCore(
		enc,
		zapcore.AddSync(writer),
		parseLevel(cfg.Level),
	)
}

// SetupPapertrailJSONIngest returns a function that ships a JSON log entry
// directly to the SolarWinds HTTP ingestion API (Papertrail's HTTP path).
//
// This transport is intentionally decoupled from zap and the global logger:
//   - No global state touched or required.
//   - No side effects on the zap pipeline.
//   - Safe to call concurrently from controllers, jobs, or goroutines.
//
// The returned function accepts a plain message string, wraps it in a
// timestamped JSON payload, and POSTs it to the ingest endpoint using
// token-based basic auth. A non-2xx response is returned as an error.
//
// Use this path when you need to ship a specific event or audit payload
// directly to Papertrail outside the structured log pipeline — for example,
// supply chain audit events or release lifecycle signals.
func SetupPapertrailJSONIngest(token string) func(msg string) error {
	const endpoint = "https://logs.collector.solarwinds.com/v1/log"

	// Client is shared across all calls to the returned function.
	// Timeout is kept short — log delivery should never block reconciliation.
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	return func(msg string) error {
		payload := map[string]interface{}{
			"message":   msg,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.SetBasicAuth("", token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 300 {
			return fmt.Errorf("papertrail ingest error: %s", resp.Status)
		}

		return nil
	}
}
