---
name: sync-docs
description: Extend the docs to match a code change (new Kind, new contract field, new provider, wiring change) in this repo and in the environments-docs site, following the shape of the existing entries. Use after adding or changing a resource type, or when asked to update, sync or regenerate docs for a change.
---

# Syncing docs with a change

Docs here list things the code defines: Kinds, fields, packages, providers,
which Kinds are wired in. When the code gains one of those, every list that
mentions its siblings needs the new entry. This skill finds those lists and
extends them. It does not rewrite or restyle what is already there.

Arguments, if given, name the change (`supplychain`, `serviceunit.route`,
a commit range). With none, document the current branch against `develop`.

## 1. Work out what changed

```bash
git fetch -q origin
git diff --name-status origin/develop...HEAD
git diff origin/develop...HEAD -- go.mod | grep blanketops
```

If the branch is `develop` itself or the diff is empty, ask which change to
document instead of guessing.

Classify each change. One branch can contain several.

| Change | How to recognise it |
|---|---|
| New Kind | New directory under `resolution/` or `pkg/apis/` |
| New or changed field | Edits to `resolution/<kind>/resolve/resolve.go` or its `Resolved*` structs |
| New provider, runtime or strategy | New file in `pkg/apis/<kind>/api/` or `strategy/`, or a new case in a selector |
| Wiring change | Edits to `resolution/contract_resolution.go` or `core/predicates/predicates.go` |
| New top-level or `pkg/` package | New directory outside the per-Kind trees |
| Contract or API bump only | `go.mod` change with no code using the new types yet |

A contract or API bump with no code using it needs no doc change. Say so
and stop.

## 2. Establish the facts from the code

Read these before writing anything. Do not carry facts over from the
previous doc text or from the change description.

| Fact | Source of truth |
|---|---|
| Field names, required or optional, conditional rules | `resolution/<kind>/resolve/resolve.go`. It wins over the `.proto` where they disagree. |
| Field types and enum values | The `.proto` in `../environments-contract/blanketops/<group>/<version>/`, at the version pinned in `go.mod` |
| Whether a Kind is wired in | `resolution/contract_resolution.go`: an uncommented field, constructor line and `Resolve` case |
| Whether a Kind has an update predicate | `core/predicates/predicates.go` |
| Phases and conditions | `pkg/apis/<kind>/domain/state.go` and the service in `pkg/apis/<kind>/application/` |
| API group and served version | `../environments-install/config/crd/bases/`, the entry with `served: true` |
| Module versions | `go.mod`, `git tag` |

If a sibling checkout is missing or its pinned version is not checked out,
say which fact could not be verified and leave that entry out. Do not fill
it from memory.

## 3. Update this repository

Find every place that lists the siblings of what changed:

```bash
grep -rn -i "<sibling>" --include='*.md' . \
  --exclude-dir=vendor --exclude-dir=docs/code --exclude=CHANGELOG.md
```

Use an existing Kind of the same sort as `<sibling>` (ServiceUnit for a new
workload Kind, Build for one with providers). Each hit that is a list, a
table or a tree is a candidate.

| Change | Places to extend |
|---|---|
| New Kind | `README.md` primitives table and Project Structure; `ROADMAP.md` API group table and its count of kinds; `docs/architecture/01-*.md` "Currently reconciled domains" and the `Adapter` listing |
| Wiring change | `docs/architecture/01-*.md` wiring sections; the "not wired" sentence at the end of `docs/DEVELOPING.md`; `README.md` if it states the Kind's status |
| New provider, runtime or strategy | `ROADMAP.md` components list; `docs/DEVELOPING.md` Patterns, only if it names the selector's implementations |
| New top-level or `pkg/` package | `README.md` Project Structure; `AGENTS.md` Layout table |
| New field | Nothing here, unless an example in these files shows the Kind's spec |

`docs/DEVELOPING.md` describes patterns, not an inventory. Change it only when a
sentence in it has become false.

Do not edit `CHANGELOG.md` (generated at release) or `docs/code/` (generated
by a workflow).

## 4. Update the docs site

The site is the `environments-docs` repository, expected at
`../environments-docs`. If it is not there, ask for its path.

```bash
git -C ../environments-docs fetch -q origin
git -C ../environments-docs status -sb
git -C ../environments-docs switch -c docs/<change> origin/develop
```

Stop and report if that checkout has uncommitted changes. Do not stash or
discard them.

| Change | Pages |
|---|---|
| New Kind | New `docs/Concepts/<kind>.md`; new `docs/Api/<Group>/<kind>.md`; the Kind's group list in `docs/Api/overview.md`; `docs/Roadmap.md` if it lists Kinds |
| New field | The field's row in `docs/Api/<Group>/<kind>.md` (and a new `#### spec.contract.<field>` table if it is an object); the Contract Semantics section and examples in `docs/Concepts/<kind>.md` |
| New enum value, provider or strategy | The value lists in the Kind's Api page; a section in its Concepts page beside the existing variants |
| New phase or condition | The status tables in the Kind's Api page; `docs/Model/state-machine.md` if it enumerates them |

`<Group>` is the capitalised API group directory: `Environments`, `Events`,
`Networks`, `Sources`. A new group needs a new directory and a new section
in `docs/Api/overview.md`.

For a new page, copy the nearest sibling page and replace its content
section by section, keeping the same headings in the same order. The Api
pages use: Description, Spec, one table per nested object, Status, phase
values, Examples, Invariants.

Sidebars are generated from the directory tree, so a new page needs no
sidebar edit. The site build fails on broken links; if `node_modules` is
present, run `npm run build` and fix what it reports.

Examples in the site use `namespace: dev` and placeholder names
(`example`, `example-org`, `*.example.com`). No real hosts, tokens or
personal data.

## 5. Writing rules

- Copy the shape of the neighbouring entry: same columns, same tense, same
  length. A new table row should be indistinguishable in form from the rows
  around it.
- Add; do not reorganise. Leave existing wording, ordering and headings
  alone unless the change made them false.
- Describe a Kind as shipped, wired or reconciled only if section 2 confirms
  it. Code that exists but is not wired in is described as not wired in.
- No emoji and no new diagrams. Where a diagram already lists the siblings,
  add the new node to it in the same style.
- In this repository, do not narrate "this repo"; give the reader the
  information.

## 6. Finish

Report, per repository:

- each file changed and what was added
- each fact that could not be verified, and the entry left out because of it
- anything found that is already wrong and unrelated to this change (list
  it; do not fix it here)

Leave both working trees uncommitted unless asked to commit. The two
repositories get separate pull requests, each against `develop`, each
using that repository's pull request template. The docs site template has
an Accuracy checklist: tick only what section 2 actually confirmed.
