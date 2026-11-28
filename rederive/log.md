## D1. Fragmented Kubernetes delivery
struggled with: initially thought of Environment as just a name/namespace
found on their own: executable versioned context that owns ClusterSecretStore and aggregates child status
did well: identified the root problem as implicit state and hidden coupling
did ok: needed constraint to make it governed not just a name
better idea:

## D2. Environment flapping on child status
struggled with: how to prevent Pending/Ready thrashing on every child status change
found on their own: ownership-based lookup with predicates, children subscribe to Environment
did well: recognized child resources reconcile at different times
did ok:
better idea:

## D3. Extensible child registration
struggled with: avoiding modification of Environment reconciler for new primitives
found on their own: declarative per-CR mediation, each CR registers itself if it wants aggregation
did well: pushed subscription logic into child reconcilers via mediation
did ok:
better idea:

## D4. Governance requires constrained objects
struggled with: how to enforce contracts without type-level definitions
found on their own: Engine + CQRS pattern, thin controllers, all CRs follow same reconciliation rules
did well: identified that governance = constrained concrete objects
did ok:
better idea:

## D5. Write-path predicate purity
struggled with: why split into Command (mutates Spec) and Observer (projects Status) paths
found on their own: writer predicates must stay pure, reader predicates must not exist
did well: recognized manager owning one controller breaks write-path determinism
did ok:
better idea:

## D6. Single-writer guarantee
struggled with: how BuildRun observer fits — it watches infrastructure, projects to Build Status.Contract
found on their own: only Command path mutates Spec; Observer never touches child Specs
did well: traced BuildRun observer code — it writes Build.Status.Contract, never Build.Spec
did ok:
better idea:

## D7. Environment as passive coordinator with own lifecycle
struggled with: why Environment needs a controller if it has no domain logic over children
found on their own: owns ClusterSecretStore reconciliation + aggregate status + own conditions/phases
did well: distinguished "lifecycle" from "domain logic"
did ok:
better idea:

## D8. No type coupling
struggled with: why not inline Environment reconciliation into each child's reconciler
found on their own: each CR owns its reconciliation (Build owns build secrets, Deployment owns deployment secrets); Environment provides ClusterSecretStore authority
did well: identified delegation and freedom per CR as the anti-coupling mechanism
did ok:
better idea:

## D9. Pull-based secret access
struggled with: why not have Environment push secrets into children's namespaces
found on their own: pull model — children fetch from convention-based key paths (blanketops/environments/git/ssh-privatekey, etc.)
did well: recognized push model breaks mediation, delegation, and resolution
did ok:
better idea:

## D10. Platform-wide conventions as declarations
struggled with: how secret key paths become platform constants without per-Environment config
found on their own: protobuf contracts own the declarations; not Go types, not K8s CRDs
did well: identified contracts as explicit declarations, not conventions
did ok:
better idea:

## D11. Three-repo separation
struggled with: why resolution logic lives in `environments` repo, not `environments-controller`
found on their own: contract (schema) → environments (resolution = "what") → controller (runtime = "how")
did well: separated concerns: bare buf contracting, resolution as reusable library, controller as runner
did ok:
better idea:

## D12. Resolution as reusable library
struggled with: why Engine + Registry dispatcher instead of direct Domain calls
found on their own: shared Engine = single dispatch/logging/error point; Registry = deterministic GVK→Domain routing; controllers stay thin
did well: recognized performance hellship if controllers call domains directly
did ok:
better idea:

## D13. Hard Command/Observer segregation
struggled with: why not let Domain.Handle also write Status or Observer mutate Spec
found on their own: deterministic rule — Spec mutations only on Command path, Status projections only on Observer path
did well: "then it becomes to know which to do when, and then it breaks the deterministic rule we are selling"
did ok:
better idea:

## D14. Dual predicate mechanism
struggled with: Domain predicates (CanCreate/CanUpdate/CanDelete) declared but never called; controller-runtime predicates do the filtering
found on their own: Domain predicates = authorization/policy ("does domain allow this"); controller predicates = event filtering ("is change meaningful")
did well: distinguished two gates at two layers
did ok:
better idea:

## D15. Domain as atomic hot-path unit
struggled with: why Engine doesn't call Domain predicates before Handle
found on their own: Engine = pure dispatcher (routing, process ownership); Domain predicates = non-leaking authorization contract; Domain cannot delegate
did well: "the domain cannot answer what, it cannot delegate, it cannot be delegated to"
did ok:
better idea:

## D16. Async pool as opt-in dual mode
struggled with: async worker pool in Engine never used (no SetWorkers calls)
found on their own: opt-in for controllers that choose it; synchronous Execute is default
did well: recognized dual mode, not dead code
did ok:
better idea:

## D17. Mediators as Domain-specific prerequisite gatherers
struggled with: why Mediators live in controller repo, not shared resolution library
found on their own: Mediators need controller-runtime client + cluster context; resolution is pure (no K8s imports)
did well: "mediators are not types, resolution are types"
did ok:
better idea:

## D18. Resolution pure, mediators have cluster access
struggled with: why resolution output is one-way projection never fed back to controllers
found on their own: resolution models domain (backend selectors, contract→domain) before controller sees it; controller stays thin
did well: recognized controllers would "eavesdrop into business that does not require it"
did ok:
better idea:

## D19. One-way projection from resolution
struggled with: why resolved intent projects to contract proto but never feeds back
found on their own: hashing, audit, gRPC — output only; controllers never inspect resolved intent
did well: kept resolution layer isolated
did ok:
better idea:

## D20. Observer reconcilers stay thin
struggled with: why observers don't reuse Domain layer for status projection
found on their own: "status projection means you no longer own the reconciliation path... let the system do what it does and take those with as they are"
did well: plain controller-runtime client.Status writes, no Engine, no Registry, no Domain
did ok:
better idea: