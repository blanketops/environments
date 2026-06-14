## What

<!-- One or two sentences. What does this PR change? -->

## Why

<!-- The intent. Link the issue if one exists: Closes #123 -->

## Domain

<!-- Mark all that apply -->

* [ ] `environments`
* [ ] `events`
* [ ] `sources`
* [ ] `networks`
* [ ] `common`
* [ ] CI / tooling / docs

## API impact

* [ ] No API surface change
* [ ] `v1alpha1` — free to change
* [ ] `v1beta1` — backwards-compatible only, deprecations allowed
* [ ] `v1` — **breaking change** (requires version bump + changelog entry + migration note)

<!-- If breaking: what breaks, and what must consumers do? -->

## Checklist

* [ ] `mage verify` passes locally
* [ ] `buf lint` clean
* [ ] `buf breaking` reviewed (failures justified above if pre-v1)
* [ ] Generated code (`buf generate`) reflects the change — all targets (Go / C# / Java / TS)
* [ ] BlanketOps labels present where required (`environments.blanketops.dev/*`)
* [ ] Docs / ESP-0001 updated if the contract semantics changed
* [ ] Commit messages follow Conventional Commits

## Notes for reviewer

<!-- Anything non-obvious: design trade-offs, follow-ups deferred, areas needing close eyes -->
