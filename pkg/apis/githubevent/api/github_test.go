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
	"context"
	"testing"

	argoeventsv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventsv1alpha1 "github.com/blanketops/environments-api/api/events/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/githubevent/domain"
	githubeventResolution "github.com/blanketops/environments/resolution/githubevent/resolve"
)

func newGitHubEventTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		corev1.AddToScheme,
		rbacv1.AddToScheme,
		argoeventsv1alpha1.AddToScheme,
		eventsv1alpha1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			t.Fatalf("AddToScheme: %v", err)
		}
	}
	return scheme
}

// TestGitHubProvider_Ensure_SetsNoOwnerReferences is a regression test for
// two compounding bugs in Ensure()'s original owner-reference stage:
//
//  1. It called SetControllerReference on every object it applied, including
//     EventBus, EventSource, and the RBAC trio — cluster-wide singletons
//     shared across every GitHubEvent CR, per Teardown's own doc comment on
//     this type. Owning them per-CR meant deleting whichever CR reconciled
//     last would cascade-delete infrastructure every other GitHubEvent CR
//     still depended on.
//  2. Worse, SetControllerReference rejects cross-namespace owner refs
//     outright, and every object it applied lives in the fixed
//     argoEventsNamespace while the CR (cr, below) can live in any
//     namespace — so in production Ensure() would fail immediately on its
//     very first SetControllerReference call for any GitHubEvent CR not
//     itself created in argoEventsNamespace, which this test's CR
//     (namespace "default") represents. This blocked the whole GitHubEvent
//     ingress feature outright, not just the singleton-ownership issue.
//
// The fix removes owner references from Ensure() entirely — Teardown
// already deletes the Sensor explicitly by its deterministic name, so no
// object here needs GC-cascade ownership at all.
func TestGitHubProvider_Ensure_SetsNoOwnerReferences(t *testing.T) {
	scheme := newGitHubEventTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	p := NewGitHubProvider(c, scheme, logr.Discard(), nil)

	cr := &eventsv1alpha1.GitHubEvent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-event", Namespace: "default", UID: "test-uid"},
	}
	resolved := &githubeventResolution.ResolvedGitHubEvent{Event: cr}
	spec := domain.GitHubEvent{
		Type:       domain.EventPush,
		Repository: domain.Repository{Owner: "acme", Name: "widgets"},
	}

	if _, err := p.Ensure(context.Background(), resolved, spec); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	// The Sensor is per-CR (deleted explicitly by Teardown) but must not
	// carry an owner ref — cr and sensor are in different namespaces, and
	// SetControllerReference would reject that combination outright.
	var sensor argoeventsv1alpha1.Sensor
	if err := c.Get(context.Background(), client.ObjectKey{
		Name: "github-sensor-my-event", Namespace: argoEventsNamespace,
	}, &sensor); err != nil {
		t.Fatalf("get sensor: %v", err)
	}
	if len(sensor.OwnerReferences) != 0 {
		t.Errorf("sensor.OwnerReferences = %+v, want none (cross-namespace CR/Sensor, cleaned up by Teardown instead)", sensor.OwnerReferences)
	}

	// The shared singletons must NOT be owned by this CR either.
	var src argoeventsv1alpha1.EventSource
	if err := c.Get(context.Background(), client.ObjectKey{
		Name: "github", Namespace: argoEventsNamespace,
	}, &src); err != nil {
		t.Fatalf("get eventsource: %v", err)
	}
	if len(src.OwnerReferences) != 0 {
		t.Errorf("EventSource (shared singleton) OwnerReferences = %+v, want none", src.OwnerReferences)
	}

	var sa corev1.ServiceAccount
	if err := c.Get(context.Background(), client.ObjectKey{
		Name: githubSensorServiceAccount, Namespace: argoEventsNamespace,
	}, &sa); err != nil {
		t.Fatalf("get serviceaccount: %v", err)
	}
	if len(sa.OwnerReferences) != 0 {
		t.Errorf("ServiceAccount (shared singleton) OwnerReferences = %+v, want none", sa.OwnerReferences)
	}

	var role rbacv1.Role
	if err := c.Get(context.Background(), client.ObjectKey{
		Name: githubEventWriterRole, Namespace: argoEventsNamespace,
	}, &role); err != nil {
		t.Fatalf("get role: %v", err)
	}
	if len(role.OwnerReferences) != 0 {
		t.Errorf("Role (shared singleton) OwnerReferences = %+v, want none", role.OwnerReferences)
	}

	var roleBinding rbacv1.RoleBinding
	if err := c.Get(context.Background(), client.ObjectKey{
		Name: githubEventWriterRoleBinding, Namespace: argoEventsNamespace,
	}, &roleBinding); err != nil {
		t.Fatalf("get rolebinding: %v", err)
	}
	if len(roleBinding.OwnerReferences) != 0 {
		t.Errorf("RoleBinding (shared singleton) OwnerReferences = %+v, want none", roleBinding.OwnerReferences)
	}
}
