package api

import (
	"context"
	"fmt"
	"time"

	kappctrlv1alpha1 "carvel.dev/kapp-controller/pkg/apis/kappctrl/v1alpha1"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ntlaletsi70/blanketops-environments/pkg/packages/domain"
	"github.com/ntlaletsi70/blanketops-environments/pkg/packages/intent"
)

type ApplicationProvider struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder events.EventRecorder // optional
}

// Compile-time contract check
var _ Provider = (*ApplicationProvider)(nil)

func NewApplicationProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	rec events.EventRecorder,
) *ApplicationProvider {
	return &ApplicationProvider{
		Client:   c,
		Scheme:   scheme,
		Log:      log,
		Recorder: rec,
	}
}

// ------------------------------------------------------------
// Execute
// ------------------------------------------------------------
func (p *ApplicationProvider) Execute(
	ctx context.Context,
	intent *intent.PackageIntent,
) (*domain.PackageResult, error) {

	start := time.Now()

	log := p.Log.WithValues(
		"package", intent.ID.Name,
		"namespace", intent.ID.Namespace,
	)

	log.Info("Executing package via kapp App")

	// ------------------------------------------------------------
	// 1. Build kapp App spec
	// ------------------------------------------------------------
	app, err := BuildKappApplication(intent)
	if err != nil {
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 2. Apply kapp App (SSA)
	// ------------------------------------------------------------
	if err := ApplyApplication(ctx, p.Client, app); err != nil {
		log.Error(err, "Failed to apply kapp App")
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 3. Observe kapp App status
	// ------------------------------------------------------------
	state, err := p.ObserveApplication(
		ctx,
		app.Namespace,
		app.Name,
	)
	if err != nil {
		log.Error(err, "Failed to observe kapp App")
		return failedResult(start, err), err
	}

	// ------------------------------------------------------------
	// 4. Build domain result
	// ------------------------------------------------------------
	return &domain.PackageResult{
		Success: false, // never final here
		Phase:   packagePhaseFromApplicationPhase(state.Phase),
		Message: state.Message,
		Kapp: domain.KappResult{
			Name:      state.Name,
			Namespace: state.Namespace,
		},
		StartedAt:  start,
		FinishedAt: time.Now(),
	}, nil
}

// ------------------------------------------------------------
// Apply
// ------------------------------------------------------------
func ApplyApplication(
	ctx context.Context,
	c client.Client,
	app *kappctrlv1alpha1.App,
) error {

	return c.Patch(
		ctx,
		app,
		client.Apply,
		&client.PatchOptions{
			FieldManager: "blanketops-packages",
			Force:        ptr.To(true),
		},
	)
}

// ------------------------------------------------------------
// Build (pure function)
// ------------------------------------------------------------
func BuildKappApplication(
	intent *intent.PackageIntent,
) (*kappctrlv1alpha1.App, error) {

	return &kappctrlv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kappctrlv1alpha1.SchemeGroupVersion.String(),
			Kind:       "App",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      intent.ID.Name,
			Namespace: intent.ID.Namespace,
		},
		Spec: kappctrlv1alpha1.AppSpec{
			// Controller-driven reconciliation
			SyncPeriod: &metav1.Duration{Duration: 0},

			Fetch: []kappctrlv1alpha1.AppFetch{
				{
					Git: &kappctrlv1alpha1.AppFetchGit{
						URL: intent.Source.RepositoryURL,
						Ref: intent.ResolvedRef,
					},
				},
			},

			Deploy: []kappctrlv1alpha1.AppDeploy{
				{
					Kapp: &kappctrlv1alpha1.AppDeployKapp{},
				},
			},
		},
	}, nil
}

// ------------------------------------------------------------
// Observe
// ------------------------------------------------------------
func (p *ApplicationProvider) ObserveApplication(
	ctx context.Context,
	namespace,
	name string,
) (*domain.ApplicationState, error) {

	var app kappctrlv1alpha1.App
	if err := p.Client.Get(
		ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		&app,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to get kapp App %s/%s: %w",
			namespace,
			name,
			err,
		)
	}

	state := &domain.ApplicationState{
		Name:      app.Name,
		Namespace: app.Namespace,
		Phase:     domain.ApplicationPhasePending,
	}

	// --------------------------------------------------------
	// Phase resolution (ReconcileSucceeded is authoritative)
	// --------------------------------------------------------
	for _, cond := range app.Status.Conditions {
		if cond.Type != "ReconcileSucceeded" {
			continue
		}

		switch cond.Status {
		case corev1.ConditionTrue:
			state.Phase = domain.ApplicationPhaseReady

		case corev1.ConditionFalse:
			state.Phase = domain.ApplicationPhaseFailed
			state.Message = cond.Message

		case corev1.ConditionUnknown:
			state.Phase = domain.ApplicationPhasePending
		}
	}

	// --------------------------------------------------------
	// Deploy execution details
	// --------------------------------------------------------
	if d := app.Status.Deploy; d != nil {

		if !d.StartedAt.IsZero() {
			t := d.StartedAt.Time
			state.DeployStartedAt = &t
		}

		if !d.UpdatedAt.IsZero() {
			t := d.UpdatedAt.Time
			state.DeployUpdatedAt = &t
		}

		state.DeployFinished = d.Finished

		if d.ExitCode != 0 {
			code := d.ExitCode
			state.DeployExitCode = &code
		}

		if d.Error != "" {
			state.Message = d.Error
		}
	}

	return state, nil
}

func packagePhaseFromApplicationPhase(
	phase domain.ApplicationPhase,
) domain.PackagePhase {

	switch phase {
	case domain.ApplicationPhaseReady:
		return domain.PackagePhaseSucceeded

	case domain.ApplicationPhaseFailed:
		return domain.PackagePhaseFailed

	case domain.ApplicationPhasePending:
		return domain.PackagePhasePending

	default:
		return domain.PackagePhaseUnknown
	}
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------
func failedResult(start time.Time, err error) *domain.PackageResult {
	return &domain.PackageResult{
		Success:    false,
		Phase:      domain.PackagePhaseFailed,
		Message:    err.Error(),
		StartedAt:  start,
		FinishedAt: time.Now(),
	}
}
