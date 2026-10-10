# Roadmap

BlanketOps Environments is pre-1.0. The APIs are `v1alpha1` and may still
change in breaking ways; [Semantic Versioning](https://semver.org/) applies,
and `v1.0.0` will mark a stable public contract.

This roadmap shows direction, not dates. Priorities are discussed in GitHub
issues and decided as described in [GOVERNANCE.md](GOVERNANCE.md). To
influence it, open an issue or comment on an existing one.

## Shipped

**Resources** — nine kinds in four API groups, each with a controller:

| API group | Kinds |
|---|---|
| `environments.blanketops.dev/v1alpha1` | Environment, Build, Deployment, ServiceUnit, Package |
| `sources.blanketops.dev/v1alpha1` | GitRepository |
| `events.blanketops.dev/v1alpha1` | GitHubEvent |
| `networks.blanketops.dev/v1alpha1` | Route, Domain |

**Components**

- Resolution engine (this repository), API types, contracts, controller,
  CLI and declarative install, each in its own repository
  ([list](CODEBASES.md)).
- Route runtimes: Knative (`knative-service`, via DomainMapping) and
  Kubernetes Ingress (`kubernetes-container`, via nginx).
- Domain TLS strategies: platform-wide DNS01 wildcard and per-domain HTTP01
  ACME, through cert-manager.
- Supply Chain plugin: Tekton, Kaniko, Trivy, Cosign and Grafeas.
- Conformance test suite that runs real Custom Resources against the
  controller binary.

**Project practice**

- CodeQL, govulncheck, OpenSSF Scorecard and native Go fuzzing in CI.

## In progress

- **Build as the reference pattern.** Make Build the worked example every
  other resource follows across the SDK.
- **API layer.** An aggregated API server is under discussion; the design is
  still being triaged.

## Planned

- **Gateway API runtime for Route.** Materialise Routes as Gateway API
  `HTTPRoute`s, alongside Knative and Ingress.
- **MCP exposure.** Drive the environment lifecycle through the Model
  Context Protocol, so AI agents can work with environments through the same
  Custom Resources people use.
- **Storage plugin.** Persistent volumes and object storage as environment
  resources.
- **API graduation.** Stabilise the `v1alpha1` APIs towards `v1beta1` and
  then `v1.0.0`.
- **SLSA provenance for release assets.** The workflow exists and is paused:
  it needs GitHub-hosted runners, which are not available to the
  organization at the moment.

## Community

- Grow the maintainer group, including maintainers from outside BlanketOps.
- Publish the SDK as an open-source repository.
- Earn the [OpenSSF Best Practices](https://www.bestpractices.dev/) badge.
- Apply for the CNCF Sandbox.
