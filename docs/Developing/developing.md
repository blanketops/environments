# Developing

This section covers development workflows for the BlanketOps Environments platform.

## Local Development

### Prerequisites

- Go 1.26.10 (toolchain specified in go.mod)
- Node.js 24 (for frontend/workflow tasks)
- Mage (for build targets)
- Kind (for local Kubernetes cluster testing)

### Building

```bash
# Standard binary
mage build

# Fully static (Gokrazy-compatible)
mage static

# Install to local bin
mage install   # → $HOME/.local/bin or $HOME/bin
```

### Testing

```bash
# Create a test cluster
kind create cluster

# Install the operator
bops-env install

# Install platform stack dependencies
bops-env dependencies install

# Verify components
kubectl get pods -A
```

### Workflow Commands

```bash
# Install the BlanketOps Environments operator
bops-env install

# Uninstall the operator
bops-env uninstall

# Update the bops-env CLI binary itself
bops-env self install
bops-env self uninstall

# Install the platform stack (dependency manifests)
bops-env dependencies install

# Manage a single dependency instead of the whole stack
bops-env dependencies list
bops-env dependencies install <name>
bops-env dependencies status [name]
bops-env dependencies uninstall <name>

# Cluster management
bops-env cluster up [name]
bops-env cluster down [name]
bops-env cluster status [name]
```

### Provenance and Verification

Every release is signed and attested to ensure artifact integrity:

- **Signed**: Binaries are signed using `cosign` (keyless mode)
- **Attested**: Every build generates a formal SLSA-compliant provenance attestation

Verification after download:

```bash
# Verify the signature
curl -LO https://github.com/blanketops/environments-cli/releases/latest/download/bops-env-static.sig
cosign verify-blob --certificate-identity-regexp ".*" --signature bops-env-static.sig bops-env-static

# Verify the attestation via GitHub CLI
gh attest verify bops-env-static --owner blanketops
```

### Contributing

Pull requests and improvements are welcome! The project is evolving toward a fully self-hosted platform bootstrap system for Kubernetes environments.