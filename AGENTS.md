# AGENTS.md

Instructions for AI coding agents working in this repository. Human
contributors should read [CONTRIBUTING.md](CONTRIBUTING.md); everything
there applies to agents too.

## What this is

A Go library (`github.com/blanketops/environments`) holding the resolution
and domain logic for BlanketOps Environments. There is no `cmd/`, no
`main.go` and no running service: the surface is the package boundary, and
the controller that calls it lives elsewhere.

Read [DEVELOPING.md](docs/DEVELOPING.md) for how the repositories relate, the
patterns in use and the steps for adding or changing a Kind. Read
[docs/architecture](docs/architecture) before changing how layers talk to
each other.

## Layout

| Path | Holds |
|------|-------|
| `resolution/<cr>/` | Pure decoding of a CR into contract types, split into `resolve`, `adapter` and `contract` |
| `pkg/apis/<cr>/` | Per-CR domain: `application` (service), `api` (Kubernetes providers), `domain` (model, state, errors) |
| `pkg/intent/` | Declared intent per resource |
| `pkg/secrets/`, `pkg/serviceaccounts/`, `pkg/providerconfig/`, `pkg/runtime/`, `pkg/utils/` | Shared platform pieces |
| `core/` | Engine, commands, conditions, events, predicates, registry, cache factory |
| `cache/` | Generation-scoped field-level object cache and typed helpers |
| `hack/` | `vendor-snapshot.sh` (CI vendor snapshots) and `build-push.sh` |
| `docs/` | `DEVELOPING.md`, `architecture/` (type system, contract boundary, engine design), and `README.md`, which says where each document lives |
| `.github/` | Workflows, templates, `CODE_OF_CONDUCT.md` |

`vendor/` is gitignored; do not commit it. `docs/code/` is generated and
committed by a workflow; do not edit it by hand.

CRD types come from `github.com/blanketops/environments-api` and contract
types from `github.com/blanketops/environments-contract`. Both are separate
modules; a change to their types is made in those repositories, not here.

## Commands

Go version is the one in [go.mod](go.mod). Lint is golangci-lint v2.

```bash
go build ./... && go vet ./...
gofmt -l .
golangci-lint run
go test ./...
```

Run all four before proposing a change. CI runs the same steps with
`-mod=vendor` against a restored vendor snapshot.

To exercise a change end to end, write a throwaway `main.go` in a scratch
directory inside the module, drive the public entrypoints against
controller-runtime's fake client, read back the objects it holds, then
delete the directory. Register status subresources on the fake client with
`WithStatusSubresource`, and check the CR's conditions rather than only the
returned error: `Reconcile` often returns the result of the status write.

## Rules

- Resolution and domain layers never panic. Return errors.
- Resolution is pure: the same spec produces the same resolved struct. No
  API server reads, clocks or randomness in `resolution/*`.
- Import contract types from
  `github.com/blanketops/environments-contract/blanketops/...`.
- Write conditions through `core/conditions.SetCondition` at each domain
  pipeline stage. Emit events through `core/events.EventRecorder` for
  terminal outcomes.
- Set BlanketOps labels (`environments.blanketops.dev/*`) where required.
- Provider writes are idempotent.
- New or changed code in `resolution/*` and `core/*` ships with tests that
  cover every statement of it.
- Keep a cohesive domain package as one package. Split only when the parts
  have independent reasons to change.

## Workflow

- Branch from `develop` and open pull requests against `develop`. `main`
  only receives releases.
- One change per pull request. Fill in
  [the pull request template](.github/PULL_REQUEST_TEMPLATE.md).
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`,
  `chore:`, `ci:`, with a scope where it helps). The changelog is generated
  from them.
- Sign off every commit with `git commit -s` (DCO).
- A change to contract semantics may need an amendment to ESP-0001. Raise it
  in an issue first.
- New workflows under `.github/workflows` end with a step that writes a
  summary to `$GITHUB_STEP_SUMMARY`.

## Do not

- Commit secrets, `.secrets`, `.vars`, `.env*`, keys or certificates.
- Report or discuss a vulnerability in a public issue. Follow
  [SECURITY.md](SECURITY.md).
- Edit `CHANGELOG.md` by hand, or bump versions outside the release
  workflows.
- State facts in docs (versions, module paths, sibling repositories) without
  checking them against `go.mod`, `git tag` or the repository itself.
