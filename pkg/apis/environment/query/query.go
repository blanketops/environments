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
// Package query provides the EnvironmentContext lookup used as step zero
// in every BlanketOps mediator.
//
// Environment must pre-exist — it is the root of the delivery chain and
// the sole authority for platform bindings (store name, store kind, type).
// All composed CRs carry environments.blanketops.dev/name and
// environments.blanketops.dev/type labels to correlate back to their
// owning Environment.
package query

import (
	"context"
	"fmt"

	environmentsv1alpha1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	environmentapi "github.com/blanketops/environments/pkg/apis/environment/api"
	environmentresolution "github.com/blanketops/environments/resolution/environment/resolve"
)

const (
	// LabelEnvironmentName is the name of the owning Environment CR.
	// Required on all BlanketOps composed CRs.
	LabelEnvironmentName = "environments.blanketops.dev/name"
	// LabelEnvironmentType is the environment classification.
	// e.g. dev, staging, production, testing
	LabelEnvironmentType = "environments.blanketops.dev/type"

	// StoreKindClusterSecretStore is the cluster-scoped ESO store kind.
	// The platform default — used whenever the Environment contract does
	// not declare a kind.
	StoreKindClusterSecretStore = "ClusterSecretStore"
	// StoreKindSecretStore is the namespaced ESO store kind.
	StoreKindSecretStore = "SecretStore"
)

// EnvironmentContext holds the resolved platform bindings for a given
// Environment CR. Returned by Lookup and passed into mediators.
type EnvironmentContext struct {
	// Name of the Environment CR.
	Name string
	// EnvironmentType from environments.blanketops.dev/type label.
	EnvironmentType string
	// StoreName is the resolved secret store name.
	// Injected into all composed CR secret reconcilers.
	StoreName string
	// StoreKind is the resolved secret store kind — ClusterSecretStore or
	// SecretStore. Defaults to ClusterSecretStore when the contract does
	// not declare one. Injected into all composed CR secret reconcilers.
	StoreKind string
}

// Lookup fetches the named Environment CR, resolves its contract, and
// returns the platform context needed by mediators.
//
// Returns a non-nil error if:
//   - environments.blanketops.dev/name label is missing
//   - Environment CR does not exist — requeue, not terminal
//   - Environment contract fails resolution
//   - Environment contract declares an unrecognized store kind
func Lookup(ctx context.Context, c client.Client, namespace string, labels map[string]string) (*EnvironmentContext, error) {
	name := labels[LabelEnvironmentName]
	if name == "" {
		return nil, fmt.Errorf("label %q is required on all BlanketOps CRs", LabelEnvironmentName)
	}
	var env environmentsv1alpha1.Environment
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &env); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("environment %q not found — must pre-exist", name)
		}
		return nil, fmt.Errorf("failed to fetch environment %q: %w", name, err)
	}
	resolved, err := environmentresolution.ResolveEnvironment(&env)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment %q: %w", name, err)
	}
	storeName := environmentapi.StoreFake
	if resolved.Spec.Contract != nil {
		storeName = environmentapi.ResolveStoreName(resolved.Spec.Contract.SecretStoreProvider)
	}
	storeKind, err := resolveStoreKind(resolved)
	if err != nil {
		return nil, fmt.Errorf("environment %q: %w", name, err)
	}
	return &EnvironmentContext{
		Name:            name,
		EnvironmentType: labels[LabelEnvironmentType],
		StoreName:       storeName,
		StoreKind:       storeKind,
	}, nil
}

// resolveStoreKind derives the secret store kind from the Environment
// contract, defaulting to ClusterSecretStore when the contract does not
// declare one. An unrecognized kind is a terminal resolution error — a
// typo here would otherwise surface much later as every ExternalSecret in
// the environment failing to bind.
func resolveStoreKind(resolved *environmentresolution.ResolvedEnvironment) (string, error) {
	kind := StoreKindClusterSecretStore
	// TODO(neo): read the declared kind from the contract once the
	// SecretStore message carries it (contract v0.4.8). Until then the
	// platform default applies unconditionally:
	// if resolved.Spec.Contract != nil && resolved.Spec.Contract.SecretStoreKind != "" {
	// 	kind = resolved.Spec.Contract.SecretStoreKind
	// }
	switch kind {
	case StoreKindClusterSecretStore, StoreKindSecretStore:
		return kind, nil
	default:
		return "", fmt.Errorf("unrecognized secret store kind %q (expected %q or %q)",
			kind, StoreKindClusterSecretStore, StoreKindSecretStore)
	}
}
