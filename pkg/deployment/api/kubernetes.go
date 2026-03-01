package api

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	//"github.com/ntlaletsi70/blanketops-environments/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8SProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

func NewK8SProvider(c client.Client, scheme *runtime.Scheme, log logr.Logger, rec record.EventRecorder) *K8SProvider {
	return &K8SProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

func (p *K8SProvider) Execute(
	ctx context.Context,
	intent *intent.DeploymentIntent,
) (*domain.DeploymentResult, error) {

	results := make([]domain.ServiceUnitResult, 0, len(intent.ServiceUnits))

	for _, su := range intent.ServiceUnits {
		res, err := p.applyServiceUnit(ctx, intent, &su)
		if err != nil {
			results = append(results, domain.ServiceUnitResult{
				Name:               su.Name,
				Phase:              domain.ServiceUnitPhase("Failed"),
				Image:              su.Image,
				Runtime:            domain.Runtime(intent.Runtime),
				Error:              err.Error(),
				LastTransitionTime: time.Now(),
			})
			continue
		}
		results = append(results, *res)
	}

	return &domain.DeploymentResult{
		Phase:          deriveDeploymentPhase(results),
		Runtime:        domain.Runtime(intent.Runtime),
		Strategy:       domain.Strategy(intent.Strategy),
		ServiceUnits:   results,
		LastUpdateTime: time.Now(),
	}, nil
}

func (p *K8SProvider) applyServiceUnit(
	ctx context.Context,
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
) (*domain.ServiceUnitResult, error) {

	if err := p.applyDeployment(ctx, intent, su); err != nil {
		return nil, err
	}

	if err := p.applyService(ctx, intent, su); err != nil {
		return nil, err
	}

	ready, err := p.isDeploymentReady(ctx, intent, su)
	if err != nil {
		return nil, err
	}

	phase := domain.ServiceUnitPhase("Deploying")
	if ready {
		phase = domain.ServiceUnitPhase("Ready")
	}

	return &domain.ServiceUnitResult{
		Name:               su.Name,
		Phase:              phase,
		Image:              su.Image,
		Runtime:            domain.Runtime(intent.Runtime),
		LastTransitionTime: time.Now(),
	}, nil
}

func (p *K8SProvider) applyDeployment(
	ctx context.Context,
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
) error {

	deploy := &appsv1.Deployment{
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

	p.Log.Info("Applying Kubernetes Deployment", "name", su.Name)

	return p.Client.Patch(
		ctx,
		deploy,
		client.Apply,
		&client.PatchOptions{
			FieldManager: "blanketops-k8s-provider",
			Force:        pointer.Bool(true),
		},
	)
}

func (p *K8SProvider) applyService(
	ctx context.Context,
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
) error {

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      su.Name,
			Namespace: intent.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"serviceUnit": su.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       su.Port,
					TargetPort: intstr.FromInt(int(su.Port)),
				},
			},
		},
	}

	p.Log.Info("Applying Kubernetes Service", "name", su.Name)

	return p.Client.Patch(
		ctx,
		svc,
		client.Apply,
		&client.PatchOptions{
			FieldManager: "blanketops-k8s-provider",
			Force:        pointer.Bool(true),
		},
	)
}

func (p *K8SProvider) isDeploymentReady(
	ctx context.Context,
	intent *intent.DeploymentIntent,
	su *intent.ServiceUnitIntent,
) (bool, error) {

	var deploy appsv1.Deployment
	if err := p.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      su.Name,
			Namespace: intent.Namespace,
		},
		&deploy,
	); err != nil {
		return false, err
	}

	if deploy.Spec.Replicas == nil {
		return false, nil
	}

	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas, nil
}

func deriveDeploymentPhase(
	results []domain.ServiceUnitResult,
) domain.DeploymentPhase {

	if len(results) == 0 {
		// Defensive default — should not normally happen
		return domain.DeploymentPhase("Pending")
	}

	allReady := true

	for _, r := range results {
		switch r.Phase {
		case domain.ServiceUnitPhase("Failed"):
			return domain.DeploymentPhase("Failed")

		case domain.ServiceUnitPhase("Ready"):
			// still possibly all ready

		default:
			allReady = false
		}
	}

	if allReady {
		return domain.DeploymentPhase("Ready")
	}

	return domain.DeploymentPhase("Deploying")
}
