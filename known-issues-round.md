# Known-Issues Round — BlanketOps Environments (deployment + teardown focus)

Status: **parked for a batched release round. NOT cutting v0.9.2 yet.** Fix across
repos, then release together. Evidence gathered on `kind-environments` with a
git-patched controller (`ko.local:readiness-gitssh-fix-git2`); findings reference
shipped code, not the patch.

Legend: repo — `C` = environments-controller, `I` = environments-install,
`CLI` = environments-cli, `APP` = for-kaniko-app-packages / app repo.

---

## Test matrix (create / teardown), evidence-based

| CR | Create | Teardown |
|----|--------|----------|
| Environment | OK (Ready once children ready) | hangs* |
| Build | OK (green after app Dockerfile fix) | hangs* |
| ServiceUnit (api/worker) | OK (worker Ready after image builds) | hangs* |
| GitRepository | Ready; webhook created; **repository MR 422** (repo exists) | hangs* + crossplane finalizer |
| GitHubEvent | OK (eventbus/eventsource/sensor created) | **confirmed hangs** (still Terminating 15s after delete) |
| Deployment | needs GH_PAT + /tmp + git; manifests repo created | **confirmed hangs**; 5 distinct blockers (below) |
| Package | OK | hangs* |
| Route | **cannot create** — strict-decoding rejects root-level `namespace` | — |
| Domain | not applied (same manifest risk) | — |

\* hangs = finalizer teardown never runs because of issue #1 (predicate). Deployment
and GitHubEvent confirmed empirically; the rest share the identical predicate path.

---

## CRITICAL — affects every finalizer-gated CR

### 1. Watch predicate drops the deletion event → teardown never runs
- **Repo:** C
- **File:** `core/predicates/predicates.go` — `MeaningfulChangePredicate().UpdateFunc` (~L58–L118)
- **Cause:** for Deployment/GitHubEvent/GitRepository/Environment/Package/ServiceUnit/Route/Domain it returns `!reflect.DeepEqual(old.Spec, newObj.Spec)`. Setting `deletionTimestamp` is **metadata-only** (spec unchanged) → returns false → update dropped. With a finalizer present no `Delete` event fires, so teardown is never triggered → object stuck `Terminating`.
- **Evidence:** Deployment and GitHubEvent both remain `Terminating` after delete; a `rollout restart` (re-list emits Create events that bypass the predicate) immediately drives the reconcile.
- **Fix:** at the top of `UpdateFunc`, before the type switch:
  ```go
  if e.ObjectOld.GetDeletionTimestamp().IsZero() && !e.ObjectNew.GetDeletionTimestamp().IsZero() {
      return true // always reconcile the deletion transition so finalizer teardown runs
  }
  ```
  Also audit finalizer-*add* paths: if a controller adds its finalizer and returns expecting the watch to re-fire, that write is metadata-only and is also dropped — return `ctrl.Result{Requeue: true}` after adding instead.

---

## Deployment CR — teardown chain (each blocks finalizer removal)

### 2. Teardown requires the Environment to pre-exist
- **Repo:** C — `internal/controller/environments/deployment.go:~149–151` (Reconcile → env lookup)
- **Error:** `environment lookup: environment "environment-sample" not found — must pre-exist`
- **Cause:** teardown resolves environment context; if the Environment was deleted first, it errors and the finalizer is never removed → stuck. Deletion-ordering dependency.
- **Fix:** teardown must tolerate a missing Environment (cache the resolved context on the Deployment, or skip env-derived lookups during teardown).

### 3. `resolveGitSSHEnv` writes the ssh key to `/tmp` (read-only rootfs)
- **Repo:** C — `pkg/apis/deployment/api/kustomization.go:392–408`
- **Error:** `write ssh key: open /tmp/...: read-only file system`
- **Fix:** go-git in-memory ssh auth (no temp file); interim: writable `/tmp` emptyDir (#9).

### 4. Clone/commit/push shell out to `git` (absent in distroless)
- **Repo:** C — `kustomization.go:428,439,442,305,329,338,689,704,709`; `pkg/utils/util.go` `RunGit`(94)/`RunGitWithEnv`(172)
- **Error:** `exec: "git": executable file not found in $PATH` (breaks create AND teardown)
- **Fix:** migrate to **go-git** (already vendored: `go-git/go-git/v5 v5.19.2` + go-billy). Keeps the distroless image; working tree still clones to `/tmp` for kustomize (keep #9). Delete `RunGit`/`RunGitWithEnv` after.

### 5. Teardown orphans the manifests repo it created
- **Repo:** C — `kustomization.go` `Teardown`(618) / `TeardownFluxResources`(572)
- **Cause:** teardown deletes the flux `GitRepository` CR (L592) and strips manifests via `removeAndPush`, but never deletes the GitHub repo created via GH_PAT.
- **Evidence:** `ntlaletsi70/for-kaniko-app-manifests` still returns HTTP 200 after teardown.
- **Fix:** settle lifecycle ownership — either delete the repo on teardown, or treat it as persistent/shared and stop creating one per Deployment (the `Teardown` doc comment already flags this ambiguity).

---

## Create-path issues

### 6. Mediators emit registry-config + webhook-secret ExternalSecrets without `property: value`
- **Repo:** C (build/deployment/githubevent mediators that generate ExternalSecrets)
- **Cause:** the bootstrapper stores Vault values wrapped as `{"value": ...}`; git-ssh refs read with `property: value` (correct), but registry-config and webhook-secret refs omit it → ESO returns the whole `{"value":...}` wrapper → invalid dockerconfigjson / wrong HMAC secret.
- **Evidence:** registry secret rendered `{"value":"{\"auths\":...}"}`; webhook secret rendered `{"value":"mysupersecret112"}`. Both only worked after I hand-patched `property: value` / stored unwrapped.
- **Fix:** emit `property: value` for these refs (or standardize scalar storage — see #14).

### 7. GitRepository tries to CREATE an existing repo → 422
- **Repo:** C (gitrepository provider → crossplane github `Repository` MR)
- **Error:** `POST /user/repos: 422 name already exists on this account`
- **Fix:** create-if-absent / adopt-existing (observe-only) when the repo already exists.

---

## environments-install (deploy manifests / RBAC — the "proper permissions", env v0.9.2)

### 8. Controller Deployment ships no `GH_PAT` env
- Blocks private manifests-repo creation (`GH_PAT must be set for private repo creation`). Wire from a Vault-backed secret.

### 9. `readOnlyRootFilesystem: true` with no writable `/tmp`
- go-git still clones a working tree to disk for kustomize → needs a writable `/tmp` emptyDir.

### 10. runAsUser / HOME / image deps
- With go-git (#4), the distroless image needs no git/ssh/passwd. Without it, the image additionally needs git, openssh-client, ca-certificates, and a non-root passwd entry + writable HOME. **Prefer go-git** to avoid all of this.

### 11. RBAC for the deployment domain
- Verify verbs for flux `GitRepository`/`Kustomization`, secrets, and crossplane MRs the deployment mediator creates/deletes.

---

## Manifest / app bugs (APP)

### 12. Root-level `namespace:` stray-indent in manifests
- `environment/route.yaml:16` has `namespace:` at root → strict-decoding CRD (Route/Domain) rejects: `unknown field "namespace"`. Non-strict CRDs (Deployment/Build) silently placed the object in the default namespace instead. Also seen in `build_buildah.yaml`, `build_buildpacks.yaml`, `deployment.yaml`, `package.yaml`.
- **Fix:** move `namespace:` under `metadata:` in every manifest.

### 13. App Dockerfile apt pin rot — FIXED
- `for-kaniko-app` Dockerfile pinned openssl on bullseye; the `.deb` was pruned (404). Bumped to `nginx:1.27-bookworm`; pushed to `main`; build green (`ghcr.io/ntlaletsi70/for-kaniko-app:main`).

---

## environments-cli (CLI)

### 14. Vault bootstrapper wraps every value as `{"value": ...}`
- Correct for opaque scalars read with `property: value`, wrong for structured/scalar secrets that mediators read property-less (registry config, webhook secret). Ties to #6.
- Already committed: env-var sourcing (`BOPS_GHCR_*`, `BOPS_GITHUB_*`), skip-placeholder, `BOPS_SSH_PRIVATE_KEY_FILE`, encrypted-key rejection. **Open:** the value-wrapping policy for structured secrets.

---

## Suggested fix sequencing for the round

1. **#1 predicate deletion-transition** — one-liner, unblocks ALL teardowns. Highest impact.
2. **#4 go-git migration** — fixes create + teardown, removes git/ssh/passwd image deps; keeps distroless.
3. **#2 teardown tolerate missing Environment**, **#5 manifests-repo lifecycle** decision.
4. **#6 mediator `property: value`** for registry/webhook; **#14 CLI** wrapping policy (paired).
5. **#7 gitrepository create-if-absent.**
6. **#8–#11 install manifests + RBAC** (cut as environments **v0.9.2**).
7. **#12 manifest namespace** fixes.

Controller release (0.6.9) carries #1, #4, #2, #5, #6, #7. Install/env (0.9.2) carries #8–#11.
