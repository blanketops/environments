# Architecture

BlanketOps Environments is designed as a self-contained Kubernetes platform bootstrapper. The architecture follows these core principles:

## Design Principles

- **Single Binary**: All manifests and bootstrap scripts are compiled into one binary via `go:embed`, eliminating filesystem dependencies at runtime
- **Zero kubectl Required**: All operations use the Kubernetes API directly via `client-go` dynamic client
- **Air-Gapped Ready**: Designed for environments where traditional tooling is unavailable — air-gapped clusters, bare metal, immutable appliances
- **Deterministic Installation Order**: The `client-go` dynamic engine ensures deterministic installation order via the Discovery REST Mapper

## Core Components

The platform stack consists of these key components in dependency order:

| Component | Role |
|-----------|------|
| **Carvel Kapp Controller** | Packaging and lifecycle management |
| **Argo Events** | Event-driven pipelines |
| **Tekton Pipelines** | CI/CD execution engine |
| **Tekton Dashboard** | Pipeline UI |
| **Shipwright Build** | Kubernetes-native image builds |
| **Crossplane** | Infrastructure orchestration |
| **External Secrets Operator** | Secure secret integration |
| **Build Strategies** | Shipwright ClusterBuildStrategy definitions |
| **Flux** | GitOps manifest sync |
| **Knative Serving** | Serverless workload runtime |
| **Kourier** | Knative's ingress/networking layer |

## Data Flow

The `client-go` dynamic engine ensures deterministic installation order:

```
CLI → Go Apply Engine → client-go Dynamic Client → Discovery REST Mapper → Kubernetes API Server → CRDs → Resources → Platform Stack
```

## Embedded Assets

All manifests are compiled into the binary via `go:embed`, and setup/teardown logic is native Go compiled straight in. Zero filesystem dependencies at runtime.

```
dependencies/     # Embedded YAML manifests (go:embed)
scripts/          # Native Go setup/teardown routines (package scripts, called by cmd)
```