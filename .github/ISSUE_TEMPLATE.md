<!--
  Pick ONE section below and delete the others.
  Title prefix: [bug] / [feature] / [architecture] / [question]
-->

# 🐛 Bug

## What happened

<!-- What did you observe? Paste exact error output, panic trace, or wrong
     reconciliation behaviour in a code block. -->

## What you expected

## Reproduce

```go
// minimal reproduction — engine setup, dispatched command/event, observed result
```

## Environment

* Module version / commit:
* Go version:
* Affected area: `<!-- core / cache / resolution / runtime / pkg/<domain> -->`
* Consumer: `<!-- which importer hit this — controller, test harness, other -->`

---

# ✨ Feature

## Problem

<!-- What can't the engine do today? Lead with the problem, not the solution. -->

## Proposed solution

<!-- Preferred approach. Sketch exported API if you have one. -->

```go

```

## Alternatives considered

## Scope

* [ ] New domain (full api / application / domain layout + registration)
* [ ] New backend provider for an existing domain
* [ ] Core engine behaviour (dispatch, predicates, registry, mutex, worker pool)
* [ ] Cache / resolution / runtime
* [ ] Tooling / CI only

---

# 🏛 Architecture

<!-- For proposed changes to the engine's structure or layering rules. -->

## Current behaviour

<!-- What the engine does / how the layers interact today. -->

## Proposed change

<!-- What changes structurally, and why the current shape is insufficient. -->

## Impact

* [ ] Layering rules unchanged (domain pure, application orchestrates, api behind provider)
* [ ] Layering rules amended — justify
* [ ] Exported API breaking — list symbols + migration path
* [ ] All ten domains affected uniformly
* [ ] Single domain only: `<!-- which -->`

---

# ❓ Question

<!-- Usage, design intent, or how the CQRS layers fit together.
     Check the docs first: https://blanketopsenvironments.netlify.app -->
