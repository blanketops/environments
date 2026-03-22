package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	environmentv1 "github.com/ntlaletsi70/blanketops-environments-api/api/environments/v1alpha1"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/intent"
	"github.com/ntlaletsi70/blanketops-environments/pkg/deployment/render/builders"
	"github.com/ntlaletsi70/blanketops-environments/pkg/utils"
)

const fieldManager = "blanketops-kustomize-provider"
const defaultGitBranch = "master"

type KustomizeStrategyProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewKustomizeStrategyProvider(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
) *KustomizeStrategyProvider {
	return &KustomizeStrategyProvider{
		Client: c,
		Scheme: scheme,
		Log:    log.WithName("kustomize-provider"),
	}
}

func (m *KustomizeStrategyProvider) BuildKustomize(
	intent *intent.DeploymentIntent,
) ([]runtime.Object, error) {

	var objects []runtime.Object

	for _, su := range intent.ServiceUnits {

		deploy := builders.BuildDeployment(intent, &su)
		svc := builders.BuildService(intent, &su)

		objects = append(objects, deploy, svc)
	}

	return objects, nil
}

//
// PUBLIC ENTRYPOINT
//

func (m *KustomizeStrategyProvider) ReconcileKustomization(
	ctx context.Context,
	cr *environmentv1.Deployment,
	intent *intent.DeploymentIntent,
	repoURL string,
	ref string,
	path string,
) error {

	if cr == nil {
		return fmt.Errorf("nil Deployment CR")
	}
	if repoURL == "" {
		return fmt.Errorf("repoURL required")
	}
	if path == "" {
		return fmt.Errorf("path required")
	}

	log := m.Log.WithValues(
		"deployment", cr.Name,
		"namespace", cr.Namespace,
	)

	log.Info("Reconciling GitOps resources")

	repoPath := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("%s-manifests", cr.Name),
	)

	env, ok := cr.Labels["environments.blanketops.dev/type"]
	if !ok || env == "" {
		return fmt.Errorf("deployment CR must define label environments.blanketops.dev/type")
	}

	// 1️⃣ Render + Commit
	if err := m.CommitAndPush(repoPath, intent, env); err != nil {
		return fmt.Errorf("commit and push failed: %w", err)
	}

	// 2️⃣ Ensure GitRepository
	if err := m.ensureGitRepository(ctx, cr, repoURL, ref); err != nil {
		return fmt.Errorf("failed ensuring GitRepository: %w", err)
	}

	// 3️⃣ Ensure Kustomization
	if err := m.ensureKustomization(ctx, cr, path); err != nil {
		return fmt.Errorf("failed ensuring Kustomization: %w", err)
	}

	log.Info("GitOps reconciliation complete")

	return nil
}

//
// INTERNALS
//

func (m *KustomizeStrategyProvider) CommitAndPush(
	repoPath string,
	intent *intent.DeploymentIntent,
	env string,
) error {

	if env == "" {
		return fmt.Errorf("environment must not be empty")
	}

	overlayPath := filepath.Join(repoPath, "overlays", env)

	if err := os.MkdirAll(overlayPath, 0755); err != nil {
		return err
	}

	// -----------------------------------------
	// 1️⃣ Clean previously generated workload files
	// -----------------------------------------

	files, err := os.ReadDir(overlayPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		name := f.Name()

		if strings.HasSuffix(name, "-deployment.yaml") ||
			strings.HasSuffix(name, "-service.yaml") {

			if err := os.Remove(filepath.Join(overlayPath, name)); err != nil {
				return err
			}
		}
	}

	// -----------------------------------------
	// 2️⃣ Generate new workload manifests
	// -----------------------------------------

	var resourceEntries []string

	for _, su := range intent.ServiceUnits {

		deploy := builders.BuildDeployment(intent, &su)
		service := builders.BuildService(intent, &su)

		// Deployment
		deployYAML, err := m.renderObject(deploy)
		if err != nil {
			return err
		}

		deployFileName := fmt.Sprintf("%s-deployment.yaml", su.Name)
		deployFile := filepath.Join(overlayPath, deployFileName)

		if err := os.WriteFile(deployFile, deployYAML, 0644); err != nil {
			return err
		}

		// Service
		serviceYAML, err := m.renderObject(service)
		if err != nil {
			return err
		}

		serviceFileName := fmt.Sprintf("%s-service.yaml", su.Name)
		serviceFile := filepath.Join(overlayPath, serviceFileName)

		if err := os.WriteFile(serviceFile, serviceYAML, 0644); err != nil {
			return err
		}

		resourceEntries = append(resourceEntries,
			deployFileName,
			serviceFileName,
		)
	}

	// -----------------------------------------
	// 3️⃣ Regenerate kustomization.yaml (deterministic)
	// -----------------------------------------

	kustFile := filepath.Join(overlayPath, "kustomization.yaml")

	var kustContent bytes.Buffer

	kustContent.WriteString(
		`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base/manifests
`,
	)

	// Preserve static environment.yaml if present
	envFile := filepath.Join(overlayPath, "environment.yaml")
	if _, err := os.Stat(envFile); err == nil {
		kustContent.WriteString("  - environment.yaml\n")
	}

	for _, r := range resourceEntries {
		kustContent.WriteString(fmt.Sprintf("  - %s\n", r))
	}

	if err := os.WriteFile(kustFile, kustContent.Bytes(), 0644); err != nil {
		return err
	}

	// -----------------------------------------
	// 4️⃣ Stage overlay directory
	// -----------------------------------------

	if _, err := utils.RunGit(repoPath, "add", filepath.Join("overlays", env)); err != nil {
		return err
	}

	// -----------------------------------------
	// 5️⃣ Commit only if changes exist
	// -----------------------------------------

	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath
	err = cmd.Run()

	if err == nil {
		m.Log.Info("No changes detected, skipping commit", "env", env)
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {

		if _, err := utils.RunGit(repoPath,
			"commit",
			"-m",
			fmt.Sprintf("render(%s): update workloads", env),
		); err != nil {
			return err
		}

		if _, err := utils.RunGit(repoPath, "push"); err != nil {
			return err
		}

		m.Log.Info("Changes committed and pushed", "env", env)
		return nil
	}

	return err
}

func (m *KustomizeStrategyProvider) renderObject(
	obj runtime.Object,
) ([]byte, error) {

	serializer := json.NewYAMLSerializer(
		json.DefaultMetaFactory,
		m.Scheme,
		m.Scheme,
	)

	var buffer bytes.Buffer

	if err := serializer.Encode(obj, &buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (m *KustomizeStrategyProvider) ensureGitRepository(
	ctx context.Context,
	cr *environmentv1.Deployment,
	repoURL string,
	ref string,
) error {

	fluxSecret := fmt.Sprintf("%s-flux-ssh", cr.Name)

	repo := &sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sourcev1.GroupVersion.String(),
			Kind:       "GitRepository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-source", cr.Name),
			Namespace: cr.Namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			Interval: metav1.Duration{Duration: 1 * time.Minute},
			URL:      repoURL,
			SecretRef: &fluxmeta.LocalObjectReference{
				Name: fluxSecret,
			},
		},
	}

	// Optional branch
	repo.Spec.Reference = &sourcev1.GitRepositoryRef{
		Branch: defaultGitBranch,
	}

	// Validate Flux secret exists (fail fast if infra broken)
	var existing corev1.Secret
	if err := m.Client.Get(
		ctx,
		client.ObjectKey{Name: fluxSecret, Namespace: cr.Namespace},
		&existing,
	); err != nil {
		return fmt.Errorf(
			"flux git secret %q not found in namespace %q: %w",
			fluxSecret,
			cr.Namespace,
			err,
		)
	}

	if err := controllerutil.SetControllerReference(cr, repo, m.Scheme); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	m.Log.Info("Applying GitRepository",
		"name", repo.Name,
		"secret", fluxSecret,
	)

	return m.Client.Patch(
		ctx,
		repo,
		client.Apply,
		&client.PatchOptions{
			FieldManager: fieldManager,
			Force:        pointer.Bool(true),
		},
	)
}

func (m *KustomizeStrategyProvider) ensureKustomization(
	ctx context.Context,
	cr *environmentv1.Deployment,
	path string,
) error {

	kust := &kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kustomizev1.GroupVersion.String(),
			Kind:       "Kustomization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kustomize", cr.Name),
			Namespace: cr.Namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			Interval:        metav1.Duration{Duration: 1 * time.Minute},
			Path:            path,
			Prune:           true,
			TargetNamespace: cr.Namespace,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: fmt.Sprintf("%s-source", cr.Name),
			},
		},
	}

	if err := controllerutil.SetControllerReference(cr, kust, m.Scheme); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	m.Log.Info("Applying Kustomization", "name", kust.Name)

	return m.Client.Patch(
		ctx,
		kust,
		client.Apply,
		&client.PatchOptions{
			FieldManager: fieldManager,
			Force:        pointer.Bool(true),
		},
	)
}
