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

package builders

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

func testIntent() *intent.DeploymentIntent {
	return &intent.DeploymentIntent{
		Name:      "web",
		Namespace: "default",
	}
}

func testServiceUnit() *serviceunitIntent.ServiceUnitIntent {
	return &serviceunitIntent.ServiceUnitIntent{
		Name:  "api",
		Image: "docker.io/blanketops/api:v1.2.3",
		Port:  8080,
		Size:  3,
	}
}

func TestBuildDeployment(t *testing.T) {
	d := BuildDeployment(testIntent(), testServiceUnit())

	if d.Name != "api" {
		t.Errorf("Name = %q, want %q", d.Name, "api")
	}
	if d.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", d.Namespace, "default")
	}
	if d.Labels["serviceUnit"] != "api" {
		t.Errorf("Labels[serviceUnit] = %q, want %q", d.Labels["serviceUnit"], "api")
	}
	if got, want := *d.Spec.Replicas, int32(3); got != want {
		t.Errorf("Replicas = %d, want %d", got, want)
	}
	if got, want := d.Spec.Selector.MatchLabels["serviceUnit"], "api"; got != want {
		t.Errorf("Selector.MatchLabels[serviceUnit] = %q, want %q", got, want)
	}
	containers := d.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(containers))
	}
	c := containers[0]
	if c.Name != "api" || c.Image != "docker.io/blanketops/api:v1.2.3" {
		t.Errorf("container = %+v, want name=api image=docker.io/blanketops/api:v1.2.3", c)
	}
	if len(c.Ports) != 1 || c.Ports[0].ContainerPort != 8080 {
		t.Errorf("container ports = %+v, want a single port 8080", c.Ports)
	}
}

func TestBuildService(t *testing.T) {
	svc := BuildService(testIntent(), testServiceUnit())

	if svc.Name != "api" {
		t.Errorf("Name = %q, want %q", svc.Name, "api")
	}
	if svc.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", svc.Namespace, "default")
	}
	if svc.Spec.Selector["serviceUnit"] != "api" {
		t.Errorf("Selector[serviceUnit] = %q, want %q", svc.Spec.Selector["serviceUnit"], "api")
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("Type = %q, want ClusterIP", svc.Spec.Type)
	}
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("len(Ports) = %d, want 1", len(svc.Spec.Ports))
	}
	p := svc.Spec.Ports[0]
	if p.Port != 8080 || p.TargetPort.IntValue() != 8080 {
		t.Errorf("port = %+v, want Port=8080 TargetPort=8080", p)
	}
}
