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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion mirrors the upjet-github provider CRD API group
// installed in the cluster. upbound/provider-github does not ship importable
// Go types — these structs own the schema locally.
var (
	SchemeGroupVersion = schema.GroupVersion{
		Group:   "repo.github.upbound.io",
		Version: "v1alpha1",
	}
	AddToScheme   = schemeBuilder.AddToScheme
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&Repository{},
		&RepositoryWebhook{},
	)
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

// ── Repository ────────────────────────────────────────────────────────────────

type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositorySpec `json:"spec"`
}

func (r *Repository) DeepCopyObject() runtime.Object {
	out := new(Repository)
	*out = *r
	out.ObjectMeta = *r.ObjectMeta.DeepCopy()
	return out
}

type RepositorySpec struct {
	ProviderConfigRef ProviderConfigRef    `json:"providerConfigRef"`
	ForProvider       RepositoryParameters `json:"forProvider"`
}

type RepositoryParameters struct {
	Name       *string `json:"name"`
	Visibility *string `json:"visibility"`
}

// ── RepositoryWebhook ─────────────────────────────────────────────────────────

type RepositoryWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositoryWebhookSpec `json:"spec"`
}

func (r *RepositoryWebhook) DeepCopyObject() runtime.Object {
	out := new(RepositoryWebhook)
	*out = *r
	out.ObjectMeta = *r.ObjectMeta.DeepCopy()
	out.Spec.ForProvider = r.Spec.ForProvider.deepCopy()
	return out
}

type RepositoryWebhookSpec struct {
	ProviderConfigRef ProviderConfigRef           `json:"providerConfigRef"`
	ForProvider       RepositoryWebhookParameters `json:"forProvider"`
}

type RepositoryWebhookParameters struct {
	Active        *bool                                      `json:"active"`
	Events        []*string                                  `json:"events"`
	Repository    *string                                    `json:"repository"`
	Configuration []RepositoryWebhookConfigurationParameters `json:"configuration"`
}

func (p RepositoryWebhookParameters) deepCopy() RepositoryWebhookParameters {
	out := p
	out.Events = append([]*string(nil), p.Events...)
	out.Configuration = append([]RepositoryWebhookConfigurationParameters(nil), p.Configuration...)
	return out
}

// RepositoryWebhookConfigurationParameters mirrors the Terraform configuration
// block. URLSecretRef sources the hook URL from a Secret — Crossplane's native
// pattern for sensitive fields.
type RepositoryWebhookConfigurationParameters struct {
	URLSecretRef SecretKeyRef `json:"urlSecretRef"`
}

// ── Shared ────────────────────────────────────────────────────────────────────

// ProviderConfigRef identifies the Crossplane ProviderConfig to use.
type ProviderConfigRef struct {
	Name string `json:"name"`
}

// SecretKeyRef is a reference to a key within a Kubernetes Secret.
type SecretKeyRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}
