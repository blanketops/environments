---
paths:
  - "pkg/apis/**"
---

# Per-Kind domain packages

- `domain/` is one cohesive package (model, state, result, errors). Do not
  split it into subpackages.
- `application/` holds the `Mapper`, the `StatusWriter` and the
  `<Kind>Service`. Only the `StatusWriter` writes status.
- Providers sit behind the `Provider` interface in `api/provider.go` and
  implement `Teardown` alongside the operation. Add a backend as a new file
  beside the existing ones and register it with the Kind's selector.
- Provider writes are idempotent: a second reconcile leaves the same
  objects.
- Objects created for a CR carry the `environments.blanketops.dev/*`
  labels.
- Set a condition with `core/conditions.SetCondition` at each pipeline
  stage, and emit an event through `core/events.EventRecorder` for a
  terminal outcome.
- `Reconcile` often returns only the status-write result. Tests assert on
  the CR's conditions, not just the returned error.
- Return errors. Expected failures are sentinel errors in
  `domain/errors.go`, wrapped with `%w`.
- Take constructor signatures from the code or the package's
  `service_test.go`, not from older notes.
