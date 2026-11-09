<p align="center">
  <img src="docs/assets/logo.png" alt="BlanketOps — Deterministic Software Delivery" width="220">
</p>

# BlanketOps Environments

**A Deterministic Software Delivery Engine for Kubernetes.**

BlanketOps Environments is the core resolution engine behind the BlanketOps platform. It provides deterministic domain logic for Kubernetes-native software delivery — transforming structured Custom Resources into stable, governed execution plans.

---

## Why BlanketOps Environments

Modern Kubernetes delivery is fragile by default.

Teams stitch together pipelines from disconnected tools — CI systems that don't know about deployments, ingress configs that don't know about certs, workload runners that don't know about environments. The result is implicit state, hidden coupling, and entropy that compounds with every release.

**BlanketOps Environments was built to eliminate that.**

Instead of pipelines, it defines delivery as a set of composable, typed domain primitives — each owning a single concern, each reconciling toward a declared intent. Every CR in the system is a first-class citizen with a well-defined lifecycle, explicit ownership boundaries, and observable state transitions.

The result is a delivery model that is:

- **Deterministic** — the same intent always produces the same outcome.
- **Observable** — every phase transition is a typed condition, not a log line.
- **Governed** — domain boundaries are enforced structurally, not by convention.
- **Composable** — primitives chain together without tight coupling.

This is not a pipeline runner. It is a reconciliation engine. The difference matters at scale.

---

## About BlanketOps Environments

BlanketOps Environments is a Kubernetes-native delivery framework designed to move code from IDE to production — with reduced entropy and governed reconciliation.

Instead of ad-hoc pipelines and implicit state, BlanketOps Environments models delivery as structured, deterministic domain primitives:

| Primitive | Responsibility | Reference |
|-----------|---------------|-----------|
| Environment | Root of the delivery chain; ClusterSecretStore authority | [docs](https://blanketops-environments.netlify.app/docs/concepts/environment/) |
| Build | Image build lifecycle; BuildRun orchestration | [docs](https://blanketops-environments.netlify.app/docs/concepts/build/) |
| Deployment | Workload rollout; ServiceUnit lifecycle | [docs](https://blanketops-environments.netlify.app/docs/concepts/deployment/) |
| Package | Artifact promotion and supply chain attestation | [docs](https://blanketops-environments.netlify.app/docs/concepts/package/) |
| GitRepository | Source binding; commit SHA resolution | [docs](https://blanketops-environments.netlify.app/docs/concepts/gitrepository/) |
| GitHubEvent | Webhook-driven trigger pipeline | [docs](https://blanketops-environments.netlify.app/docs/concepts/githubevent/) |
| ServiceUnit | Single workload declaration (image, port, size) | [docs](https://blanketops-environments.netlify.app/docs/concepts/serviceunit/) |
| Route | Workload-to-host binding; runtime materialisation | [docs](https://blanketops-environments.netlify.app/docs/concepts/route/) |
| Domain | TLS chain ownership; cert-manager + Knative bridge | [docs](https://blanketops-environments.netlify.app/docs/concepts/domain/) |

---

## Core Flow Principle

All inputs — whether from SDKs, YAML, or event systems — are normalized into a single source of truth:

> **Custom Resources (CRDs)**

From there:

1. Controllers observe state changes.
2. The resolver (this repository) normalizes and validates intent.
3. The engine executes deterministically.
4. Status is reconciled back into the resource.

This ensures:

- Consistent behaviour across all clients.
- Deterministic execution.
- Observable state transitions.
- No hidden side effects.

---

## Documentation

The full BlanketOps Environments documentation is available at:

[blanketops-environments.netlify.app](https://blanketops-environments.netlify.app)

| Reference | Link |
|-----------|------|
| CRD Definitions (Build API) | [docs](https://blanketops-environments.netlify.app/docs/api/environments/build/) |
| API Overview & State Transitions | [docs](https://blanketops-environments.netlify.app/docs/api/overview/) |
| Delivery Lifecycle (State Machine Model) | [docs](https://blanketops-environments.netlify.app/docs/model/state-machine/) |

---

## Design Principles

BlanketOps Environments follows strict architectural boundaries:

- **Deterministic resolution** — the same spec always produces the same resolved struct.
- **Explicit domain modelling** — every CR owns exactly one concern. Nothing bleeds.
- **Clear state transitions** — every phase is typed; every condition is owned.
- **Separation of domain and infrastructure** — resolution is pure Go; providers are Kubernetes.
- **Convention over coordination** — `ksvc name == ServiceUnit name` eliminates cross-domain status reads.
- **Cache as fast path** — field-level generation-scoped cache eliminates redundant API server reads across the reconciliation hot path.
- **No hidden side effects** — provider dispatch is idempotent; `CreateOrUpdate` everywhere.

The goal is to reduce delivery entropy through structured reconciliation.

---

## Stability

| | |
|---|---|
| **Current Version** | `v0.8.8` |
| **API Status** | Evolving — breaking changes possible before `v1.0.0` |
| **Intended Use** | Integration with BlanketOps Environments controllers |
| **Versioning** | Semantic Versioning — `v1.0.0` will signal a stable public contract |

---

## Community

BlanketOps Environments is developed in the open, and contributions are welcome.

| | |
|---|---|
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to report issues, set up a development environment and open pull requests |
| [DEVELOPING.md](docs/DEVELOPING.md) | How the repositories relate, the patterns the code follows, and how to add or change a resource type |
| [CODE_OF_CONDUCT.md](.github/CODE_OF_CONDUCT.md) | The CNCF Code of Conduct, which applies in all project spaces |
| [CODEBASES.md](CODEBASES.md) | The repositories that make up the project and what each is for |
| [GOVERNANCE.md](GOVERNANCE.md) | How decisions are made and how the project relates to BlanketOps' commercial products |
| [MAINTAINERS.md](MAINTAINERS.md) | Who maintains the project |
| [Roadmap](https://blanketops-environments.netlify.app/docs/Roadmap) | What has shipped and what is planned |
| [ADOPTERS.md](ADOPTERS.md) | Who uses it — add your organization |
| [SECURITY.md](SECURITY.md) | Supported versions and how to report a vulnerability privately |

---

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
