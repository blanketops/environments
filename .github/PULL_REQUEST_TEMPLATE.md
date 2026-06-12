<!--
  Title must follow Conventional Commits — git-cliff builds the changelog from it.
  e.g. feat(deployment): add gitops decorator for kustomize intents
       fix(core): guard engine mutex on concurrent dispatch
       chore(resolution): rename contract adapter
-->

## What

<!-- One or two sentences. What does this PR change? -->

## Why

<!-- The intent. Link the issue if one exists: Closes #123 -->

## Area

<!-- Mark all that apply -->

* [ ] `core` — engine, registry, dispatch, predicates, conditions
* [ ] `cache` — object cache, adapters
* [ ] `resolution` — contract adapters / resolvers
* [ ] `runtime` — event dispatch, trigger matching
* [ ] `pkg/<domain>` — which: `<!-- build / buildtrigger / deployment / domain / githubevent / gitrepository / packages / routes / serviceunit -->`
* [ ] `secrets` / `serviceaccounts` / `logging`
* [ ] CI / tooling / docs

## Exported API impact

<!-- This is a library — the controller and others import it. -->

* [ ] No exported API change
* [ ] Additive — new exported symbols, backwards compatible
* [ ] **Breaking** — exported symbols removed / signatures changed (requires minor bump pre-v1, migration note below)

<!-- If breaking: which symbols, and what must importers do? -->

## Architecture

<!-- The layering is the contract. Confirm it held. -->

* [ ] `domain` stays pure — no imports from `application` or `api`
* [ ] `application` orchestrates — backends selected via `backend_selector`, no provider types leaked upward
* [ ] `api` providers stay behind the `provider.go` interface
* [ ] New domains follow the full layout (api / application / domain) and register in `core` + `resolution` + `cache`
* [ ] N/A — no structural change

## Checklist

* [ ] `go build ./...` and `go test ./...` pass
* [ ] `go vet ./...` clean
* [ ] `staticcheck ./...` clean (suppressions via `staticcheck.conf` only)
* [ ] `go mod tidy` produces no diff
* [ ] Commit messages follow Conventional Commits

## Notes for reviewer

<!-- Anything non-obvious: design trade-offs, follow-ups deferred, areas needing close eyes -->
