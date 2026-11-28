# Decision map
repo: environments | built from: 6cf32f2 | date: 2026-11-26
root problem: Kubernetes delivery is a fragmented mess of disconnected tools that don't know about each other, creating implicit state and hidden coupling that compounds with every release.

## D1. Fragmented delivery — executable Environment context
- Problem: tools (CI, ingress, workloads, environments) don't know about each other; implicit state, hidden coupling
- Given: Kubernetes clusters, CRDs, controllers, protobuf
- Constraints, in order:
  1. Must eliminate implicit state → breaks: ad-hoc pipeline stitching
  2. Must enforce governed context → breaks: everyone defining Environment differently
  3. Must be executable/versioned → breaks: static YAML references
- Decision: Environment as executable versioned context owning ClusterSecretStore + aggregate status
- Options: (a) static namespace, (b) Environment CR with controller, (c) external registry — (b) wins: first-class CR with lifecycle
- Evidence: commit b628ec9 "first commit", f9a1959 "Environments #, first commit stubs"
- Where: core/environment/environment.go — Environment CRD definition

## D2. Child status flapping — ownership-based lookup with predicates
- Problem: Environment thrashes Pending/Ready on every child status change
- Given: children reconcile at different times, can fail independently
- Constraints, in order:
  1. Must reflect true aggregate readiness → breaks: naive polling
  2. Must be inexpensive → breaks: full relist on every child event
  3. Children own their lifecycle → breaks: Environment pushing status
- Decision: ownership-based lookup + predicate filtering; children subscribe to Environment
- Options: (a) push from children, (b) pull on schedule, (c) event-driven with predicates — (c) wins
- Evidence: commit 6e959d0 "ci(release): sign with verifiable Sigstore bundles", buildrun observer pattern
- Where: internal/controller/observers/environment/environment.go — fan-in via Watches + EnqueueRequestsFromMapFunc

## D3. Extensible child registration — declarative per-CR mediation
- Problem: adding new primitive shouldn't require modifying Environment reconciler
- Given: Build, Deployment, ServiceUnit, Package, GitRepository, GitHubEvent, Route, Domain
- Constraints, in order:
  1. New types register themselves → breaks: central registration
  2. No shared mediator → breaks: coupling via common library
  3. Declaration not assignment → breaks: imperative registration calls
- Decision: declarative per-CR mediation; each CR defines its own mediator if it wants aggregation
- Options: (a) central registry, (b) shared mediator, (c) per-CR mediator — (c) wins
- Evidence: internal/mediators/* per domain, each Domain constructs its own in SetupWithManager
- Where: internal/domains/*/environment.go — Domain constructs Mediator

## D4. Governance requires constrained objects — Engine + CQRS + thin controllers
- Problem: governance/determinism impossible if concrete objects have too much freedom
- Given: CQRS pattern, Engine, Registry, Domain interface
- Constraints, in order:
  1. All CRs follow same reconciliation rules → breaks: ad-hoc per-CR logic
  2. Controller must be thin → breaks: fat controllers with business logic
  3. Deterministic dispatch → breaks: direct Domain calls
- Decision: shared Engine + Registry dispatcher; thin controllers route via Command
- Options: (a) direct Domain calls, (b) Engine dispatch, (c) async queue only — (b) wins
- Evidence: core/engine/engine.go, core/registry/registry.go, internal/runtime/runtime.go
- Where: core/engine/engine.go — Execute() routes via Registry

## D5. Write-path predicate purity — Command/Observer hard segregation
- Problem: writer predicates must stay pure, reader predicates must not exist
- Given: single manager, controller-runtime predicates
- Constraints, in order:
  1. Spec mutations only on Command path → breaks: Observer mutating Spec
  2. Status projections only on Observer path → breaks: Domain.Handle writing Status
  3. Single-writer guarantee per resource → breaks: multiple writers
- Decision: hard segregation — Command path (Domain.Handle mutates Spec), Observer path (plain client.Status writes)
- Options: (a) unified reconciler, (b) segregated paths — (b) wins
- Evidence: architecture doc 02 "CQRS Delegated Domain Engine", buildrun observer pattern
- Where: internal/controller/observers/buildrun/buildrun.go — writes Build.Status.Contract, never Build.Spec

## D6. Single-writer guarantee — BuildRun observer pattern
- Problem: Build domain's Ensure() only reports Triggered=true; doesn't wait for BuildRun to finish
- Given: Shipwright BuildRun, Build CR, contract status
- Constraints, in order:
  1. Only Command path mutates Spec → breaks: observer writing Spec
  2. Observer watches infrastructure (BuildRun) → breaks: watching Build CR
  3. Outcome projected to Status.Contract → breaks: writing to Spec
- Decision: BuildRun observer watches BuildRun, resolves owner via label, writes Build.Status.Contract + Conditions
- Options: (a) Build domain waits for BuildRun, (b) separate observer — (b) wins
- Evidence: internal/controller/observers/buildrun/buildrun.go
- Where: internal/controller/observers/buildrun/buildrun.go — Reconcile() exits for non-terminal, writes Status.Contract

## D7. Environment as passive coordinator with own lifecycle
- Problem: Environment needs controller but has no domain logic over children
- Given: ClusterSecretStore, aggregate status, conditions/phases
- Constraints, in order:
  1. Must reconcile ClusterSecretStore → breaks: static secret refs
  2. Must aggregate child status → breaks: no visibility
  3. Must have own lifecycle (conditions) → breaks: not a living CR
- Decision: Environment controller exists for ClusterSecretStore + aggregate status + own conditions; no child domain logic
- Options: (a) inline into children, (b) separate controller — (b) wins
- Evidence: internal/controller/environments/environment.go, observer pattern
- Where: internal/controller/environments/environment.go — reconciles ClusterSecretStore, aggregates via fan-in

## D8. No type coupling — each CR owns its reconciliation
- Problem: inlining Environment logic into children breaks ownership
- Given: Build owns build secrets, Deployment owns deployment secrets, etc.
- Constraints, in order:
  1. Each CR single non-overlapping domain → breaks: shared reconciliation logic
  2. Environment provides ClusterSecretStore authority → breaks: children managing secrets independently
  3. Delegation and freedom per CR → breaks: central orchestrator
- Decision: Environment provides store ref; children pull secrets via mediation
- Options: (a) Environment injects secrets, (b) children pull — (b) wins
- Evidence: pkg/secrets/* reconcilers, ExternalSecret key conventions
- Where: pkg/secrets/registry/build/build.go — Build mediator pulls registry config

## D9. Pull-based secret access — convention-based key paths
- Problem: push model breaks mediation, delegation, resolution
- Given: ClusterSecretStore, ExternalSecrets, standardized keys
- Constraints, in order:
  1. Children fetch from store Environment authorizes → breaks: Environment writing to child namespaces
  2. Platform-wide conventions, not per-Environment config → breaks: custom key paths per Environment
  3. Declarations, not conventions → breaks: implicit naming
- Decision: children pull from convention-based keys (blanketops/environments/git/ssh-privatekey, blanketops/environments/registry/config)
- Options: (a) per-Environment keys, (b) platform constants — (b) wins
- Evidence: secure-software-supply-chain skill, ExternalSecret key format rules
- Where: pkg/secrets/* — all reconcilers use same key paths

## D10. Platform-wide conventions as declarations — protobuf contracts
- Problem: conventions without enforcement are implicit, not declarative
- Given: protobuf schemas, independent versioning (v1alpha1 → v1beta1 → v1)
- Constraints, in order:
  1. Contracts own declarations, not Go types → breaks: kubebuilder markers as truth
  2. Independent versioning from K8s CRD → breaks: tied to API server machinery
  3. Append-only once stable → breaks: breaking changes in stable versions
- Decision: protobuf contracts as canonical truth; CRD is opaque envelope (runtime.RawExtension)
- Options: (a) CRD OpenAPI as truth, (b) protobuf contracts — (b) wins
- Evidence: environments-contract repo, architecture doc 01 "Type System & Contract Boundary"
- Where: blanketops/networks/v1alpha1/route.proto — canonical message definition

## D11. Three-repo separation — contract / resolution / controller
- Problem: resolution logic cluttering contract or controller repos
- Given: bare buf contracting, resolution as reusable library, controller as runner
- Constraints, in order:
  1. Contract = bare schema only → breaks: resolution logic in contract repo
  2. Resolution = reusable library (no cmd/main) → breaks: compiled into controller
  3. Controller = runtime only → breaks: business logic in controller
- Decision: environments-contract (protobuf), environments (resolution library), environments-controller (runtime)
- Options: (a) monorepo, (b) two repos, (c) three repos — (c) wins
- Evidence: three repos in BlanketOps org, go.mod replace directives
- Where: environments/go.mod — replace github.com/blanketops/environments => ../environments

## D12. Resolution as reusable library — shared Engine + Registry
- Problem: direct Domain calls create performance hellship, non-deterministic dispatch
- Given: Engine, Registry, Domain interface, command.Command
- Constraints, in order:
  1. Single dispatch/logging/error point → breaks: each caller handling differently
  2. Deterministic GVK→Domain routing → breaks: ad-hoc registration
  3. Controllers stay thin → breaks: fat controllers
- Decision: shared Engine + Registry dispatcher; controllers build Command, call Engine.Execute
- Options: (a) direct calls, (b) Engine dispatch, (c) async only — (b) wins
- Evidence: core/engine/engine.go, core/registry/registry.go, internal/runtime/runtime.go
- Where: core/engine/engine.go — Execute() looks up Domain by GVK, calls Handle

## D13. Hard Command/Observer segregation — deterministic reconciliation
- Problem: letting Domain.Handle write Status or Observer mutate Spec breaks determinism
- Given: two segregated code paths, environment observer package doc
- Constraints, in order:
  1. Spec mutations only Command path → breaks: Observer writing Spec
  2. Status projections only Observer path → breaks: Domain.Handle writing Status
  3. "CQRS boundary: this reconciler never touches composed CR specs" → breaks: observer mutating children
- Decision: hard segregation enforced by repo convention and doc comments
- Options: (a) unified, (b) soft segregation, (c) hard segregation — (c) wins
- Evidence: architecture doc 02, internal/controller/observers/environment/environment.go
- Where: internal/controller/observers/environment/environment.go — "CQRS boundary" doc comment

## D14. Dual predicate mechanism — Domain predicates vs controller predicates
- Problem: Domain predicates (CanCreate/CanUpdate/CanDelete) declared but never called; controller predicates do filtering
- Given: core/domain/domain.go, core/predicates/predicates.go
- Constraints, in order:
  1. Domain predicates = authorization ("does domain allow this") → breaks: using for event filtering
  2. Controller predicates = event filtering ("is change meaningful") → breaks: using for authorization
  3. Domain cannot leak authorization decisions → breaks: delegating to policy service
- Decision: two different gates at two layers; Domain predicates part of Domain contract, controller predicates at controller-runtime level
- Options: (a) single predicate, (b) dual — (b) wins
- Evidence: architecture doc 02 "Known issue: two predicate mechanisms"
- Where: core/domain/domain.go — interface; core/predicates/predicates.go — MeaningfulChangePredicate

## D15. Domain as atomic hot-path unit — Engine as pure dispatcher
- Problem: Engine calling Domain predicates would leak domain knowledge into dispatcher
- Given: Engine, Domain interface, hot path
- Constraints, in order:
  1. Engine = pure dispatcher (routing, process ownership) → breaks: Engine knowing domain rules
  2. Domain predicates = non-leaking authorization contract → breaks: Domain delegating
  3. Hot path atomic, untainted → breaks: adding indirection
- Decision: Engine calls Handle directly; Domain predicates exist as contract but called by nothing (yet)
- Options: (a) Engine calls predicates, (b) Engine pure — (b) wins
- Evidence: core/engine/engine.go Execute() does not call CanCreate/CanUpdate/CanDelete
- Where: core/engine/engine.go — Execute() only registry lookup + Handle call

## D16. Async pool as opt-in dual mode
- Problem: async worker pool in Engine never used (no SetWorkers calls)
- Given: Queue, SetWorkers, StartWorkers, 1024-capacity channel
- Constraints, in order:
  1. Controllers choose their mode → breaks: forced async
  2. Synchronous Execute default → breaks: async-only
  3. Dual mode, not dead code → breaks: removing it
- Decision: opt-in async for controllers that need it; sync is default
- Options: (a) sync only, (b) async only, (c) dual mode — (c) wins
- Evidence: core/engine/engine.go — SetWorkers/StartWorkers exist but unwired
- Where: core/engine/engine.go — workerLoop, Queue, SetWorkers, StartWorkers

## D17. Mediators as Domain-specific prerequisite gatherers
- Problem: shared mediator registry couples domains; mediators need cluster access
- Given: controller-runtime client, cluster lookups, secrets, Git SHAs
- Constraints, in order:
  1. Mediators need cluster access → breaks: putting in pure resolution library
  2. Each Domain knows its prerequisites → breaks: generic mediator
  3. Mediators are compositions, not simple objects → breaks: shared service
- Decision: each Domain constructs its own Mediator composition in SetupWithManager
- Options: (a) shared mediator registry, (b) per-Domain mediator — (b) wins
- Evidence: internal/mediators/*, internal/domains/*/environment.go
- Where: internal/domains/build/environment.go — constructs buildMediator

## D18. Resolution pure, mediators have cluster access
- Problem: resolution reaching into cluster breaks purity, testability, portability
- Given: resolution layer (contract-typed), mediator layer (cluster-typed)
- Constraints, in order:
  1. Resolution = pure Go, no K8s imports → breaks: cluster access in resolution
  2. Mediators = full controller-runtime client → breaks: putting in resolution
  3. Resolution output = one-way projection → breaks: feeding back to controllers
- Decision: resolution never reaches cluster; mediators prepare runtime prerequisites
- Options: (a) resolution reaches cluster, (b) strict split — (b) wins
- Evidence: resolution/* imports no k8s.io/*; mediators import controller-runtime
- Where: resolution/route/resolve/resolve.go — pure Go; internal/mediators/gitRepository/ — cluster access

## D19. One-way projection from resolution — never fed back
- Problem: controllers inspecting resolved intent breaks thin controller principle
- Given: resolved domain intent → contract proto (hashing, audit, gRPC)
- Constraints, in order:
  1. Projection = output only → breaks: feeding back to controllers
  2. Controllers never inspect resolved intent → breaks: thick controllers
  3. Contract proto = canonical serialization → breaks: Go structs as truth
- Decision: resolution projects one-way to contract proto; never fed back to controllers
- Options: (a) feed back, (b) output only — (b) wins
- Evidence: resolution/route/contract/contract_adapter.go doc comment
- Where: resolution/route/contract/contract_adapter.go — "This projection is NEVER fed back into controllers"

## D20. Observer reconcilers stay thin — plain client.Status
- Problem: reusing Domain layer for status projection breaks reconciliation ownership
- Given: observer pattern, plain controller-runtime client
- Constraints, in order:
  1. Observer never touches Spec → breaks: Domain.Handle writing Status
  2. "Status projection means you no longer own the reconciliation path" → breaks: reusing Domain
  3. Let system do what it does, take results as they are → breaks: aggregating incompleteness
- Decision: observers are plain controller-runtime clients writing Status; no Engine, no Registry, no Domain
- Options: (a) reuse Domain, (b) thin observers — (b) wins
- Evidence: internal/controller/observers/* — all use plain client.Client
- Where: internal/controller/observers/buildrun/buildrun.go — client.Client + StatusWriter, no Engine