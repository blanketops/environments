---
paths:
  - "resolution/**"
---

# Resolution

- `Resolve<Kind>` only decodes and validates. No client calls, no clock, no
  randomness: the same spec yields the same struct.
- Reject a nil object and an empty `Spec.Contract` with an error. Never
  panic.
- Keep the three packages per Kind apart: `resolve` (decode), `adapter`
  (injection wrapper), `contract` (one-way projection to the proto). The
  projection is output only and is never fed back into reconciliation.
- Enums and field semantics come from
  `github.com/blanketops/environments-contract/blanketops/...`. Do not
  redeclare them.
- Every statement you add or change is covered by a test in the same
  package.
- A change to a `resolve.go` keeps that Kind's `FuzzResolve<Kind>` passing.
  A new Kind adds one, plus a matrix entry in
  `.github/workflows/fuzzing.yml`.
- A new Kind is not reachable until it has a field, a constructor line and
  a `Resolve` case in `resolution/contract_resolution.go`.
