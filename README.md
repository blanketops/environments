# 🌍 BlanketOps Environments

**A Deterministic Software Delivery Engine for Kubernetes**

BlanketOps Environments is the core resolution engine behind the BlanketOps Environments platform.
It provides deterministic domain logic for Kubernetes-native software delivery — transforming structured Custom Resources into stable, governed execution plans.

This repository contains **pure domain and resolution logic only**.
It does not run standalone and does not include controllers or transport layers.

---

## 🧭 About BlanketOps

BlanketOps is a Kubernetes-native delivery framework designed to move code from IDE to production in minutes — with reduced entropy and governed reconciliation.

Instead of ad-hoc pipelines and implicit state, BlanketOps Environments models delivery as structured, deterministic domain primitives:

| Primitive | Reference |
|-----------|-----------|
| Builds | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/build) |
| Deployments | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/deployment) |
| Packages | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/build) |
| Git Repositories | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/gitrepository) |
| Service Units | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/serviceunit) |
| Event Triggers | [docs](https://blanketopsenvironments.netlify.app/docs/Concepts/githubevent) |

This repository implements the **resolution engine** that powers those primitives.

---

## 🧠 What This Repository Is

This module is:

* A pure Go library
* Infrastructure-agnostic
* Transport-agnostic
* Deterministic by design
* Intended to be embedded in controllers or services

It is designed to be imported by:

* Kubernetes controllers
* Delivery operators
* API backends
* CLI tools
* Platform orchestration layers

> **This module is not intended to be executed directly.**

---

## 🧩 How It Fits Into BlanketOps Environments

BlanketOps Environments operates as the **resolution layer** within a larger control plane.

Different clients express intent in different ways, but all converge into a shared reconciliation model.

```mermaid
flowchart TD
    subgraph Clients
        A1["SDK Client"]
        A2["YAML · kubectl"]
        A3["Events · MCP"]
    end

    subgraph API
        B1["gRPC API Server"]
    end

    subgraph Kubernetes
        C1["Custom Resources — CRDs"]
    end

    subgraph Control Loop
        D1["Controller · Reconciler"]
        D2["Domain — This Repository"]
        D3["Execution Engine"]
    end

    A1 --> B1
    A3 --> B1
    A2 --> C1
    B1 --> C1
    C1 --> D1
    D1 --> D2
    D2 --> D3
    D3 --> C1
```

---

## 🌐 BlanketOps Environments Ecosystem

BlanketOps Environments is composed of multiple repositories, each with a clear responsibility.

This diagram shows how they relate — from the protobuf contract layer, through the core domain engine, into the platform runtime, and out to the client interfaces.

```mermaid
graph TB
    %% ── Source of Truth ──
    CONTRACT["📜 environments-contract\nProtobuf definitions · Buf codegen\nSource of truth for all service contracts"]

    %% ── Core Domain Engine ──
    CORE["🧠 environments\nPure domain engine · Resolution logic\nDeterministic · Transport-agnostic\nBuild · Deploy · Package · GitRepo · ServiceUnit · Route"]

    %% ── Platform Layer ──
    CONTROLLER["⚙️ environments-controller\nK8s reconciliation loops\nThin controllers · Reconciliation-first"]

    API["🔌 environments-api\ngRPC transport layer\nService definitions from contract"]

    SUPPLYCHAIN["🔒 environments-supply-chain\nSecurity envelope operator\nTekton · Trivy · Cosign · Grafeas\nSupplyChain CR · ImageBuild CR"]

    %% ── Client Layer ──
    SDK["📦 environments-sdk\nMulti-language SDK\n.NET · Java · TypeScript"]

    CLI["💻 environments-cli\nTerminal UX\nOnboarding · Install flow"]

    MCP["🤖 environments-mcp\nAI-native interface\nModel Context Protocol server"]

    CLIENTSET["🔧 environments-clientset\nTyped Go client\nGenerated from CRD definitions"]

    KUBECONFIG["🔑 environments-kubeconfig\nCluster auth & context management"]

    KUBECTL["⌨️ kubectl-blanketops\nkubectl plugin\nNative K8s CLI integration"]

    %% ── Quality ──
    TESTS["🧪 environments-tests\nEnd-to-end integration tests\nCross-repo validation"]

    %% ── Relationships ──

    CONTRACT -->|"proto definitions"| CORE
    CONTRACT -->|"gRPC service defs"| API
    CONTRACT -->|"Buf codegen"| SDK
    CONTRACT -->|"schema contracts"| MCP

    CORE -->|"domain logic"| CONTROLLER
    CORE -->|"resolution engine"| API
    CORE -->|"build domain"| SUPPLYCHAIN

    API -->|"gRPC endpoints"| SDK
    API -->|"gRPC endpoints"| CLI
    API -->|"gRPC endpoints"| MCP
    API -->|"gRPC endpoints"| CLIENTSET
    API -->|"gRPC endpoints"| KUBECTL

    CONTROLLER -->|"reconciles CRs"| SUPPLYCHAIN

    CLIENTSET -->|"auth provider"| KUBECONFIG
    CLI -->|"auth provider"| KUBECONFIG
    KUBECTL -->|"auth provider"| KUBECONFIG

    TESTS -.->|"validates"| CONTROLLER
    TESTS -.->|"validates"| API
    TESTS -.->|"validates"| SUPPLYCHAIN
    TESTS -.->|"validates"| CLI

    %% ── Styling ──

    classDef contract fill:#FAEEDA,stroke:#854F0B,color:#412402,stroke-width:2px
    classDef core fill:#EEEDFE,stroke:#534AB7,color:#26215C,stroke-width:2px
    classDef platform fill:#E1F5EE,stroke:#0F6E56,color:#04342C,stroke-width:1px
    classDef client fill:#FAECE7,stroke:#993C1D,color:#4A1B0C,stroke-width:1px
    classDef quality fill:#F1EFE8,stroke:#5F5E5A,color:#2C2C2A,stroke-width:1px

    class CONTRACT contract
    class CORE core
    class CONTROLLER,API,SUPPLYCHAIN platform
    class SDK,CLI,MCP,CLIENTSET,KUBECONFIG,KUBECTL client
    class TESTS quality
```

---

## 🔁 Core Flow Principle

All inputs — whether from SDKs, YAML, or event systems — are normalized into a single source of truth:

> **Custom Resources (CRDs)**

From there:

1. Controllers observe state changes.
2. The resolver (this repository) normalizes and validates intent.
3. The engine executes deterministically.
4. Status is reconciled back into the resource.

This ensures:

* Consistent behaviour across all clients
* Deterministic execution
* Observable state transitions
* No hidden side effects

---

## 📚 Documentation

The full BlanketOps Environments documentation is available at:

🔗 [blanketopsenvironments.netlify.app](https://blanketopsenvironments.netlify.app)

| Reference | Link |
|-----------|------|
| 📘 CRD Definitions (Build API) | [docs](https://blanketopsenvironments.netlify.app/docs/Api/Environments/build) |
| 📘 API Overview & State Transitions | [docs](https://blanketopsenvironments.netlify.app/docs/Api/overview) |
| 🧠 Delivery Lifecycle (State Machine Model) | [docs](https://blanketopsenvironments.netlify.app/docs/Model/state-machine) |

---

## 📦 Installation

```bash
go get github.com/ntlaletsi70/blanketops-environments@v0.1.9
```

---

## 🏗 Project Structure

```
core/           → Engine and orchestration logic
pkg/            → Domain modules (build, deployment, packages, etc.)
resolution/     → Resolution contracts and adapters
runtime/        → Event runtime components
logging/        → Structured logging abstractions
```

---

## ✨ Design Principles

BlanketOps Environments follows strict architectural boundaries:

* Deterministic resolution
* Explicit domain modeling
* Clear state transitions
* Separation of domain and infrastructure
* Reproducible outcomes
* No hidden side effects

The goal is to reduce delivery entropy through structured reconciliation.

---

## 🚧 Stability

| | |
|---|---|
| **Current Version** | `v0.1.9` |
| **API Status** | Evolving — breaking changes may occur |
| **Intended Use** | Alpha integration with BlanketOps Environments controllers |
| **Versioning** | Semantic Versioning — `v1.0.0` will signal a stable public contract |

---

## 🎯 Intended Consumers

This module powers:

* BlanketOps Environments Controllers
* Delivery orchestration layers
* Reconciliation engines

If you are looking for the controller runtime, see the BlanketOps Environments Controller repository.

---

## 🤝 Contributing

This project is currently in active development.
Contributions and architectural discussions are welcome.

---

## 📜 License

Apache License 2.0 — see [LICENSE](LICENSE) for details.