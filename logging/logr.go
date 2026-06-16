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

package logging

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// AsLogr wraps a zap.Logger as a logr.Logger using the zapr bridge.
// This is the adapter that makes zap compatible with controller-runtime,
// which requires a logr.Logger via ctrl.SetLogger.
//
// Called by Init after buildZap succeeds. Can also be used directly when
// a component needs a logr.Logger derived from a custom zap instance
// (e.g. a child logger with additional fields pre-attached).
func AsLogr(z *zap.Logger) logr.Logger {
	return zapr.NewLogger(z)
}
