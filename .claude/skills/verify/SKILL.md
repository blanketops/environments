---
name: verify
description: Project-specific recipe for driving this repo's code end-to-end (no cmd/, no running service)
---

# Verifying blanketops/environments

This repo is a pure Go domain/resolution library (`resolution/`, `pkg/apis/`,
`pkg/intent/`, `cache/`, `core/`) — no `cmd/`, no `main.go`, no running
service. The real controller lives in a separate operator repo this
workspace doesn't contain. So "run the app" isn't literal; the surface is
the package boundary.

## Recipe: throwaway `go run` driver, not `_test.go`

Write a small `main.go` under a scratch dir **inside the module** (so it
resolves via `vendor/` — a driver outside the repo won't see the same
`go.mod`), importing the real public entrypoints exactly as an external
caller would (e.g. `pkg/apis/deployment/application.NewDeploymentService`),
and drive it with a real `sigs.k8s.io/controller-runtime/pkg/client/fake`
client seeded with the CR(s) under test. Run with `go run ./scratchdir/`,
read the actual objects the fake client ends up holding, delete the
scratch dir when done — don't commit it, don't leave it as a `_test.go`.

Example shape (Deployment CR, verified 2026-07-18):

```go
scheme := runtime.NewScheme()
environmentv1alpha1.AddToScheme(scheme)
appsv1.AddToScheme(scheme)
corev1.AddToScheme(scheme)

c := fake.NewClientBuilder().
    WithScheme(scheme).
    WithObjects(deploymentCR).
    WithStatusSubresource(&environmentv1alpha1.Deployment{}). // see gotcha below
    Build()

svc := deploymentapp.NewDeploymentService(
    deploymentIntent.NewIntentBuilder(),
    deploymentapp.NewStatusWriter(c, logr.Discard()),
    deploymentapi.NewReconciliationExecutor(
        deploymentapi.NewRuntimeProvider(c, scheme, logr.Discard(), nil),
        deploymentapi.NewKustomizeStrategyProvider(c, scheme, logr.Discard()),
        logr.Discard(),
    ),
    logr.Discard(),
)
err := svc.Reconcile(ctx, resolvedDeployment, resolvedServiceUnits, logr.Discard())
// then client.Get the objects it should have created and print them
```

## Gotchas learned the hard way

- **`WithStatusSubresource(&Type{})` is required**, not optional, on this
  controller-runtime version (v0.24.1) for any type with a status
  subresource. Omitting it makes `Status().Update()` fail with a confusing
  `"<kind> \"<name>\" not found"` error instead of a clear one — looks like
  the object doesn't exist when it actually does.
- **Server-side apply (`client.Apply` + `Force: true`) works fine against
  the fake client** on this version — confirmed via `K8SProvider.applyDeployment`/
  `applyService`'s `Patch(ctx, obj, client.Apply, &client.PatchOptions{...})`.
- **The fake client has no real Deployment controller**, so
  `status.readyReplicas` never populates — anything that gates on readiness
  (e.g. `K8SProvider.isDeploymentReady`) will correctly report "not ready."
  Don't mistake that for a bug when driving Deployment/K8s-workload flows.
- **`Reconcile`'s Go error return often only reflects the *status write*,
  not the underlying operation.** `DeploymentService.Reconcile` (and the
  same shape elsewhere, e.g. Route) derives conditions from the operation's
  result/error, writes those conditions to the CR, and returns whatever the
  *status write* returned — a failed reconciliation still returns `nil` if
  the CR update itself succeeded. Check CR conditions, not just the
  returned `error`, to see what actually happened.
