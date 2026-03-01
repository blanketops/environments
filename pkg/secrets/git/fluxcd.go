package git

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	deploymentResolution "github.com/ntlaletsi70/blanketops-environments/resolution/deployment"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const githubKnownHosts = "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

type DeploymentFluxGitSSHSecretReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func NewDeploymentFluxGitSSHSecretReconciler(
	c client.Client,
	log logr.Logger,
) *DeploymentFluxGitSSHSecretReconciler {
	return &DeploymentFluxGitSSHSecretReconciler{
		Client: c,
		Log:    log,
	}
}

func (r *DeploymentFluxGitSSHSecretReconciler) Reconcile(
	ctx context.Context,
	deployment *deploymentResolution.ResolvedDeployment,
) error {
	secretName := fmt.Sprintf("%s-flux-ssh", deployment.Deployment.Name)
	namespace := deployment.Deployment.Namespace

	// Check if secret already exists — if so, leave it alone.
	// We never rotate keys automatically; the keypair is stable for
	// the lifetime of the Deployment CR.
	existing := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Name:      secretName,
		Namespace: namespace,
	}, existing)

	if err == nil {
		// Already exists — nothing to do, key is stable
		r.Log.V(1).Info(
			"Flux Git SSH secret already exists, skipping",
			"deployment", deployment.Deployment.Name,
			"secret", secretName,
		)
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get flux ssh secret: %w", err)
	}

	// ── Secret does not exist — generate a fresh unique keypair ──
	r.Log.Info(
		"Generating new Flux Git SSH keypair",
		"deployment", deployment.Deployment.Name,
		"secret", secretName,
	)

	privateKeyPEM, publicKeyAuthorized, err := generateSSHKeypair(
		fmt.Sprintf("flux-%s", deployment.Deployment.Name),
	)
	if err != nil {
		return fmt.Errorf("generate ssh keypair: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"blanketops.dev/managed":    "true",
				"blanketops.dev/purpose":    "flux-git-ssh",
				"blanketops.dev/deployment": deployment.Deployment.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"identity":     privateKeyPEM,
			"identity.pub": publicKeyAuthorized,
			"known_hosts":  []byte(githubKnownHosts),
		},
	}

	if err := controllerutil.SetControllerReference(
		deployment.Deployment,
		secret,
		r.Client.Scheme(),
	); err != nil {
		return fmt.Errorf("set owner reference: %w", err)
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		return fmt.Errorf("create flux ssh secret: %w", err)
	}

	r.Log.Info(
		"Flux Git SSH secret created",
		"deployment", deployment.Deployment.Name,
		"secret", secretName,
		"publicKey", string(publicKeyAuthorized),
	)

	return nil
}

// PublicKey returns the authorized_keys format public key from an existing secret.
// Used by ensureDeployKey to register the key on GitHub.
func (r *DeploymentFluxGitSSHSecretReconciler) PublicKey(
	ctx context.Context,
	deployment *deploymentResolution.ResolvedDeployment,
) (string, error) {
	secretName := fmt.Sprintf("%s-flux-ssh", deployment.Deployment.Name)
	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      secretName,
		Namespace: deployment.Deployment.Namespace,
	}, secret); err != nil {
		return "", fmt.Errorf("get flux ssh secret: %w", err)
	}

	pub, ok := secret.Data["identity.pub"]
	if !ok {
		// Fall back to deriving it from the private key
		signer, err := ssh.ParsePrivateKey(secret.Data["identity"])
		if err != nil {
			return "", fmt.Errorf("parse private key: %w", err)
		}
		return string(ssh.MarshalAuthorizedKey(signer.PublicKey())), nil
	}

	return string(pub), nil
}

// generateSSHKeypair generates a fresh ed25519 keypair.
// Returns (privateKeyPEM, publicKeyAuthorizedKeys, error).
func generateSSHKeypair(comment string) ([]byte, []byte, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Marshal private key to OpenSSH PEM format
	pemBlock, err := ssh.MarshalPrivateKey(privKey, comment)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(pemBlock)

	// Marshal public key to authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("derive public key: %w", err)
	}
	publicKeyAuthorized := ssh.MarshalAuthorizedKey(sshPubKey)

	return privateKeyPEM, publicKeyAuthorized, nil
}

// Ensure the spec drift check still works for anything calling the old unstructured path.
// This is a no-op shim so callers don't break during transition.
func specChanged(a, b map[string]any) bool {
	return !reflect.DeepEqual(a, b)
}
