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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	intent "github.com/BlanketOps/blanketops-environments/pkg/intent/deployment"
)

func BuildDeployment(
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
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

func BuildService(
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
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
