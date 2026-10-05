---
name: verify
description: Pre-commit verification for a code change in this repo — the CI gates, the per-change checks (coverage, fuzz, wiring) and, for behaviour changes, driving the code against a fake client. Run before committing or opening a pull request for anything that touches Go files.
---

# Verifying a change

There is no binary to launch: the module is a library, and the surface is
the package boundary. Verifying means passing the same gates CI runs, then
the checks specific to what changed, then observing the changed behaviour.

Docs-only and workflow-only changes skip this skill.

## 1. Scope the change

```bash
git status -s
git diff --name-only origin/develop...HEAD -- '*.go' go.mod go.sum
```

Note which trees are touched: `resolution/`, `core/`, `pkg/apis/<kind>/`,
`cache/`, `go.mod`. Sections 3 and 4 depend on it.

## 2. Gates (always)

Run all four. They are the required CI checks.

```bash
go build ./... && go vet ./...
gofmt -l .
golangci-lint run --timeout=5m
go test ./...
```

`gofmt -l` must print nothing. CI runs the same steps with `-mod=vendor`
against a restored snapshot; there is no `vendor/` locally and none should
be committed.

If `go build` fails resolving `github.com/blanketops/*`, that is private
module access (`GOPRIVATE`, git credentials), not the change.

## 3. Checks by what changed

| Touched | Check |
|---|---|
| `resolution/*` or `core/*` | Every function added or changed is fully covered (below) |
| `resolution/<kind>/resolve/` | Run that Kind's fuzz target (below) |
| A new Kind | Wiring (below) |
| `go.mod` | `go mod tidy` leaves `go.mod` and `go.sum` unchanged |
| Any non-test file in `resolution/` or `pkg/apis/*/domain` | No new `panic(` |
| Objects created by a provider | They carry the `environments.blanketops.dev/*` labels |
| A pipeline stage or terminal outcome | Condition set via `core/conditions.SetCondition`; event via `core/events.EventRecorder` |

Coverage:

```bash
go test ./resolution/... ./core/... -coverprofile=/tmp/c.out -covermode=atomic
go tool cover -func=/tmp/c.out | awk '$3 != "100.0%"'
```

This lists every function below 100%. None of the functions the change
added or edited may appear in it. The trees are not at 100% overall (98.8%
on 2026-10-05, with gaps in the engine worker pool and ServiceUnit
resolution), so other entries are existing gaps: leave them unless the
change touches that function, and do not report the tree as fully covered.
A file of only type and const declarations has no statements and needs no
test.

Fuzz, 30 seconds is enough locally:

```bash
go test ./resolution/<kind>/resolve/... -run=FuzzResolve<Kind> -fuzz=FuzzResolve<Kind> -fuzztime=30s
```

A new Kind needs its own `FuzzResolve<Kind>` in `resolve_test.go` and a
matrix entry in `.github/workflows/fuzzing.yml`.

Wiring for a new Kind:

```bash
grep -n -i '<kind>' resolution/contract_resolution.go core/predicates/predicates.go
```

Expect a field, a constructor line and a `Resolve` case in the first file,
and a `case` in `MeaningfulChangePredicate` in the second. If the Kind is
deliberately left unwired, say so in the pull request.

## 4. Observe the behaviour

Passing tests show the tests pass. For a change to what a service or
provider does, run it and look at the result.

1. Create a scratch directory inside the module (for example
   `./zz_verify/`) with a `main.go`. It must be inside the module to
   resolve the same dependencies.
2. Build the Kind's service the way the controller would. Take the
   constructor calls from the package's own `service_test.go`; do not copy
   them from older notes, the signatures move.
3. Use `sigs.k8s.io/controller-runtime/pkg/client/fake`, seeded with the CR
   under test. Register each type that has a status subresource with
   `WithStatusSubresource`.
4. Call `Reconcile` (or the provider method), then `Get` the CR and every
   object it should have created, and print them.
5. `go run ./zz_verify/`, read the output, then delete the directory.

What to look for:

- **Conditions on the CR, not the returned error.** `Reconcile` commonly
  returns the result of the status write, so a failed operation can return
  `nil`.
- **Run it twice.** The second run must leave the same objects: no
  duplicates, no error.
- **Teardown**, where the Kind has one: the objects are gone afterwards.

Known limits of the fake client:

- Omitting `WithStatusSubresource` makes status updates fail with
  `"<kind> \"<name>\" not found"` although the object exists.
- No controllers run, so readiness fields (`status.readyReplicas` and the
  like) never populate. Anything gated on readiness reports not ready.
- Server-side apply with `Force` works.

## 5. Report

State, with the command output behind each:

- the four gates: pass or fail
- each check from section 3 that applied, and its result
- what section 4 showed, or that the change had no behaviour to observe
- anything skipped, and why

Do not describe a step as passing if it was not run. If the change adds or
alters a Kind, field, provider or wiring, run the `sync-docs` skill next.
