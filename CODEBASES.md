# Codebases

BlanketOps Environments is developed across the repositories below, all
under the Apache License 2.0. Together they are the project: its
[governance](GOVERNANCE.md), [security policy](SECURITY.md) and
[Code of Conduct](.github/CODE_OF_CONDUCT.md) apply to every one of them.

| Repository | Purpose |
|---|---|
| [environments](https://github.com/blanketops/environments) | Resolution engine — turns Custom Resources into execution plans (this repository) |
| [environments-api](https://github.com/blanketops/environments-api) | Kubernetes API types and CRD definitions |
| [environments-contract](https://github.com/blanketops/environments-contract) | Canonical contracts shared by every component |
| [environments-controller](https://github.com/blanketops/environments-controller) | Kubernetes controller that runs the reconciliation loops |
| [environments-cli](https://github.com/blanketops/environments-cli) | Command-line client |
| [environments-install](https://github.com/blanketops/environments-install) | Declarative installation (CRDs and controller manifests) |
| [environments-tests](https://github.com/blanketops/environments-tests) | Conformance test suite for the API surface |
| [environments-docs](https://github.com/blanketops/environments-docs) | Documentation website |
| [secure-software-supplychain](https://github.com/blanketops/secure-software-supplychain) | Supply Chain plugin (Tekton, Kaniko, Trivy, Cosign, Grafeas) |

How the repositories depend on one another, and which one a change belongs
in, is described in [DEVELOPING.md](docs/DEVELOPING.md).
