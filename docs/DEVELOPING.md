# Developing BlanketOps Environments

How the code is organised, how the repositories fit together, and how to
make the common kinds of change. For process — issues, branches, commit
format, sign-off, review — see [CONTRIBUTING.md](../CONTRIBUTING.md).

## How the repositories relate

A resource such as a Build or a ServiceUnit is defined once, as a versioned
protobuf contract. Kubernetes stores it inside an opaque envelope. This
module turns that envelope into typed intent and supplies the logic to act
on it. The controller runs that logic.

| Repository | Owns | Relation to this module |
|---|---|---|
| [environments-contract](https://github.com/blanketops/environments-contract) | Protobuf schemas under `blanketops/<group>/<version>/*.proto` and their generated Go | Imported here. The meaning of every field comes from it. |
| [environments-api](https://github.com/blanketops/environments-api) | CRD Go types and generated CRD manifests | Imported here. Every Kind is an envelope: `Spec.Contract runtime.RawExtension` plus `Status.Conditions`. |
| environments | Resolution, per-Kind domain logic, providers, the `core/` engine framework, the cache | This module. A library: no `cmd/`, no manager. |
| [environments-controller](https://github.com/blanketops/environments-controller) | The running manager: reconcilers, `core/domain.Domain` implementations, mediators | Imports this module and calls into it. |
| [environments-install](https://github.com/blanketops/environments-install) | Installable manifests: CRDs, RBAC, samples | Carries copies of the CRDs generated in environments-api. Not a Go dependency. |
| [environments-cli](https://github.com/blanketops/environments-cli) | The `bops` command-line client | Not a Go dependency. Shipped alongside in releases. |
| [environments-tests](https://github.com/blanketops/environments-tests) | Conformance test suite for the API surface | Not a Go dependency. |
| [environments-docs](https://github.com/blanketops/environments-docs) | The documentation website | Describes the Kinds and their state transitions. |

Go dependencies point one way:

```
environments-contract ─┐
                       ├─→ environments ─→ environments-controller
environments-api ──────┘
```

A change that crosses repositories lands in that order, and each step is a
tagged release that the next one picks up through `go.mod`. Dependabot opens
the bump pull requests here for environments-api and environments-contract.

Pinned versions are in [go.mod](../go.mod). Check them before assuming a
contract field or a CRD Kind is available.

## Path of a reconcile

1. The controller receives an event for a CR and builds a
   `core/command.Command` (type, GVK, object).
2. It calls `core/engine.Engine.Execute`. The engine looks the GVK up in
   `core/registry.Registry` and calls `Handle` on the registered
   `core/domain.Domain`.
3. The `Domain` implementation lives in the controller. It calls the
   resolution adapter for the Kind.
4. `resolution/<kind>/resolve` decodes `Spec.Contract`, validates it and
   returns a `Resolved<Kind>` struct.
5. The Domain passes that struct to the Kind's service in
   `pkg/apis/<kind>/application`.
6. The service maps it to the domain model, runs any providers from
   `pkg/apis/<kind>/api`, derives conditions from the outcome and writes
   them to the CR's status.

`core/` defines the framework (`Command`, `Domain`, `Registry`, `Engine`)
and implements none of the Domains. Everything else in this module is a
building block a Domain draws on.

The controller also runs observers: reconcilers that watch child resources
and write status only. They do not go through `Command` or the engine.
[docs/architecture](architecture) covers both sides in detail.

## Anatomy of a Kind

Each Kind has the same footprint. ServiceUnit is the smallest complete
example and a good one to copy from.

| Path | Purpose |
|---|---|
| `resolution/<kind>/resolve/` | `Resolve<Kind>(cr)` and the `Resolved<Kind>` structs. Decoding and validation only. |
| `resolution/<kind>/adapter/` | `Adapter` with a `Resolve` method wrapping the function above, so callers can inject it. |
| `resolution/<kind>/contract/` | One-way projection from the resolved struct to the contract proto, for hashing and audit. Never fed back into reconciliation. |
| `resolution/contract_resolution.go` | Top-level `Adapter` that dispatches any supported CR to its Kind's adapter. |
| `pkg/apis/<kind>/domain/` | The model: `model.go`, `state.go` (phases), `result.go`, `errors.go` (sentinel errors). |
| `pkg/apis/<kind>/application/` | `Mapper` (resolved to domain), `StatusWriter`, and the `<Kind>Service` with `Reconcile`. |
| `pkg/apis/<kind>/api/` | Providers that create real cluster objects. Present only when the Kind materialises something. |
| `pkg/intent/<kind>/` | Declared-intent builders, where a Kind needs them. |
| `cache/<kind>/` | Typed cache wrapper with `Set*`/`Get*` helpers and `PublishResolved`. |
| `core/predicates/predicates.go` | The Kind's case in `MeaningfulChangePredicate`. |

Deployment adds `reconcile/`, `strategy/` and `render/` because it dispatches
across runtimes. Add extra layers only when a Kind has that need.

## Patterns

**Contract first.** Field names, enums and semantics come from the
contract. Import its types from
`github.com/blanketops/environments-contract/blanketops/...`. Do not
redeclare a contract enum locally.

**Resolution decodes and nothing else.** A `Resolve<Kind>` function takes
the CR, rejects a nil object or an empty `Spec.Contract`, decodes the raw
JSON, validates required fields and returns a typed struct or an error. The
same spec always yields the same struct. No client calls, no clock, no
randomness.

**Three packages per Kind in `resolution/`.** `resolve`, `adapter` and
`contract` change for different reasons and are kept apart.

**One cohesive `domain` package.** Model, phases, result and errors refer
to each other and stay in one package.

**Service, Mapper, StatusWriter.** The service is constructed with its
collaborators and exposes `Reconcile`. The Mapper converts resolved to
domain types. The StatusWriter is the only thing that writes status.

**Providers behind an interface.** A Kind that materialises objects defines
a `Provider` interface in `api/provider.go` with an operation and a
`Teardown`. Implementations sit beside it (`kaniko.go`, `buildah.go`,
`buildpacks.go` for Build). Selection is explicit: Build uses
`application.BackendSelector`, Deployment uses `api.ProviderRegistry`.

**Idempotent writes.** Providers apply or create-or-update, so a repeated
reconcile converges and does not duplicate.

**Errors are returned.** Resolution and domain code does not panic.
Expected failures are sentinel errors in `domain/errors.go`, wrapped with
`%w` for context.

**Conditions carry the outcome.** Set them with
`core/conditions.SetCondition` at each stage. A service's `Reconcile` often
returns only the result of the status write, so a failed operation can
still return `nil`: read the CR's conditions to know what happened.

**Events for terminal outcomes.** Emit them through
`core/events.EventRecorder`.

**The cache is a fast path.** `cache.ObjectCache` stores fields keyed by
namespace, name and generation. Each Kind's wrapper publishes its resolved
fields so other Kinds can read them without an API call. A miss or a
backend error falls through to the informer cache, so publishing is
best-effort and must not fail a reconcile.

**Labels.** Objects created on behalf of a CR carry the
`environments.blanketops.dev/*` labels.

## Running and testing

```bash
go build ./... && go vet ./...
gofmt -l .
golangci-lint run
go test ./...
```

Tests sit next to the code as `_test.go` files and use controller-runtime's
fake client. Register every type that has a status subresource with
`WithStatusSubresource`, or status updates fail with a misleading "not
found". The fake client runs no controllers, so readiness fields such as
`status.readyReplicas` are never populated.

New or changed code in `resolution/*` and `core/*` ships with tests that
cover every statement of it.

To see a change work end to end, write a throwaway `main.go` in a scratch
directory inside the module, build the Kind's service as the controller
would, run it against a fake client seeded with a CR, and print the objects
the client ends up holding. Delete the directory afterwards.

## Changing an existing Kind

Adding or changing a field:

1. Add the field to the Kind's `.proto` in environments-contract and
   release it. Skip this if the field already exists there.
2. Bump environments-contract in [go.mod](../go.mod).
3. Decode and validate the field in `resolution/<kind>/resolve/resolve.go`
   and add it to the `Resolved<Kind>Spec` struct.
4. Add it to the projection in `resolution/<kind>/contract/`.
5. Carry it through the `Mapper` into `pkg/apis/<kind>/domain`, and into
   providers if it affects what gets created.
6. Add `Set*`/`Get*` helpers in `cache/<kind>/` and include it in
   `PublishResolved` if other Kinds need to read it.
7. Add tests for the accepted and rejected forms of the field.

The CRD does not change: the envelope stores the contract without parsing
it.

## Adding a new Kind

Work through the repositories in dependency order. Steps 1, 2, 4 and 5 are
summaries; follow each repository's own contributor and agent guides for
the exact commands.

1. **environments-contract.** Add
   `blanketops/<group>/v1alpha1/<kind>.proto` with the Kind's message, spec
   and status. Shared enums go under `blanketops/common/v1`. Regenerate
   with `mage gen`, check with `mage verify`, and release.

2. **environments-api.** Copy an existing Kind file in
   `api/<group>/v1alpha1/` and rename it. Keep the envelope shape; do not
   add typed spec fields. Run `mage generate`, `mage manifests` and
   `mage verify`, then release.

3. **environments** (here).
   1. Bump both modules in [go.mod](../go.mod).
   2. Create `resolution/<kind>/resolve`, `adapter` and `contract`, with
      tests.
   3. Wire the adapter into
      [resolution/contract_resolution.go](../resolution/contract_resolution.go):
      a field on `Adapter`, a line in `NewAdapter`, a case in `Resolve`.
   4. Create `pkg/apis/<kind>/domain` and `pkg/apis/<kind>/application`.
      Add `pkg/apis/<kind>/api` with a `Provider` interface if the Kind
      creates cluster objects.
   5. Create `cache/<kind>/` if other Kinds will read its resolved fields.
   6. Add a case for the Kind to `MeaningfulChangePredicate` in
      [core/predicates/predicates.go](../core/predicates/predicates.go). A
      Kind with no case reconciles on every update, including status-only
      ones.
   7. Add the Kind to the primitives table and Project Structure in
      [README.md](../README.md).
   8. Release.

4. **environments-controller.** Bump this module. Add a Domain under
   `internal/domains/<kind>/`, a mediator under
   `internal/mediators/<kind>/`, and a reconciler under
   `internal/controller/<group>/`. Register the scheme and the reconciler
   in `internal/bootstrap/register.go`, add RBAC markers and run
   `mage manifests`.

5. **environments-install.** Add the generated CRD, its RBAC roles and a
   sample manifest.

6. **environments-docs.** Add the concept page and the API reference.

A Kind whose code exists here but is not wired into
`resolution/contract_resolution.go` and not registered in the controller
is not reconciled. Route and Domain are in that state today.

A change to what a contract means, as opposed to adding to it, may need an
amendment to ESP-0001. Open an issue first.
