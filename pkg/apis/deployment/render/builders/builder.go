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
Package builders owns BuildDeployment and BuildService — pure functions
that translate a DeploymentIntent and a single ServiceUnitIntent into
Kubernetes object literals. No client, no side effects: both the imperative
path (pkg/apis/deployment/api.K8SProvider, which applies these objects live)
and the GitOps path (pkg/apis/deployment/api.KustomizeStrategyProvider,
which renders them into committed manifests) share this same construction
logic, so the two paths can't silently drift apart on what a ServiceUnit's
Kubernetes objects look like.
*/
package builders

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	intent "github.com/blanketops/environments/pkg/intent/deployment"
	serviceunitIntent "github.com/blanketops/environments/pkg/intent/serviceunit"
)

// BuildDeployment renders su's Kubernetes Deployment object literal within
// intent. Pure — no client, no side effects.
func BuildDeployment(
	intent *intent.DeploymentIntent,
	su *serviceunitIntent.ServiceUnitIntent,
) *appsv1.Deployment {

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      su.Name,
			Namespace: intent.Namespace,
			Labels: map[string]string{
				"serviceUnit": su.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &su.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"serviceUnit": su.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"serviceUnit": su.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  su.Name,
							Image: su.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: su.Port},
							},
						},
					},
				},
			},
		},
	}
}

// BuildService renders su's Kubernetes Service object literal within
// intent. Pure — no client, no side effects.
func BuildService(
	intent *intent.DeploymentIntent,
	su *serviceunitIntent.ServiceUnitIntent,
) *corev1.Service {

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      su.Name,
			Namespace: intent.Namespace,
			Labels: map[string]string{
				"serviceUnit": su.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"serviceUnit": su.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       su.Port,
					TargetPort: intstr.FromInt(int(su.Port)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
