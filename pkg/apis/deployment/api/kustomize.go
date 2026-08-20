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

package api

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KustomizationProvider holds the shared clients used to construct a
// KustomizeStrategyProvider.
type KustomizationProvider struct {
	Client                       client.Client
	Scheme                       *runtime.Scheme
	Log                          logr.Logger
	NewKustomizeStrategyProvider *KustomizeStrategyProvider
}

// NewKustomizationProvider constructs a KustomizeStrategyProvider from the
// given clients.
func NewKustomizationProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, Recorder record.EventRecorder) *KustomizeStrategyProvider {
	return &KustomizeStrategyProvider{
		Client: c,
		Scheme: scheme,
		Log:    log,
	}
}
