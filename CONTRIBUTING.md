# Contributing to BlanketOps Environments

Thank you for your interest. Contributions of every kind are welcome: bug
reports, documentation, reviews, design discussion and code.

Everyone taking part follows the [Code of Conduct](.github/CODE_OF_CONDUCT.md). How
decisions are made is described in [GOVERNANCE.md](GOVERNANCE.md).

## Where things live

BlanketOps Environments spans several repositories; the list is in
[GOVERNANCE.md](GOVERNANCE.md#scope). Open issues and pull requests in the
repository whose code you want to change. If you are not sure which one,
open an issue here.

## Reporting a bug or asking for a feature

Open an issue using one of the templates — **Bug**, **Feature** or
**Question**. For a bug, include the exact error output, what you expected,
and the smallest set of commands or Custom Resources that reproduces it.

Do **not** report security vulnerabilities in a public issue. Follow
[SECURITY.md](.github/SECURITY.md) instead.

## Proposing a larger change

For a change to a public API, a contract or the delivery model, open an
issue first describing the problem and the proposed approach, so it can be
discussed before you invest in code. Changes to contract semantics may also
need an amendment to ESP-0001, the Environment Standardization Proposal; the
feature template asks about this.

## Development setup

You need:

- Go — the version in [`go.mod`](go.mod)
- [golangci-lint](https://golangci-lint.run/) v2

```bash
git clone https://github.com/blanketops/environments.git
cd environments
go build ./...
go test ./...
```

The architecture notes in [`docs/architecture`](docs/architecture) explain
the type system, the contract boundary and how the engine is split across
repositories.

[DEVELOPING.md](docs/DEVELOPING.md) covers how the repositories relate, the
patterns the code follows, and how to add or change a resource type.
[docs/README.md](docs/README.md) says where each document lives.

## Making a change

1. Fork the repository and create a branch from **`develop`**.
   `main` only receives releases.
2. Make your change, with tests. Resolution and core packages are expected
   to stay fully covered.
3. Run the checks the pull request template asks for:

   ```bash
   go build ./... && go vet ./...
   gofmt -l .
   golangci-lint run
   go test ./...
   ```

4. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
   (`feat:`, `fix:`, `docs:`, `chore:` …, with a scope where it helps, e.g.
   `feat(domain): …`). The changelog is generated from these messages.
5. Sign off every commit (see [below](#developer-certificate-of-origin)).
6. Open a pull request against `develop` and fill in the template.

## Design rules for code changes

The [design principles](README.md#design-principles) in the README are
enforced in review. In particular:

- Resolution and domain layers do not panic; errors are returned.
- Contract types are imported from
  `github.com/blanketops/environments-contract/blanketops/...`.
- Conditions are written through `core/conditions.SetCondition` at each
  domain pipeline stage, and terminal outcomes emit events through
  `core/events.EventRecorder`.
- BlanketOps labels (`environments.blanketops.dev/*`) are present where
  required.

## Review and merge

A maintainer reviews every pull request. The required CI checks must pass:
**Vendor Snapshot** and **Lint, Vet, Test**, plus **Test and Coverage** for
release merges into `main`. Expect review feedback; once approved, a
maintainer merges.

## Developer Certificate of Origin

Contributions are accepted under the
[Developer Certificate of Origin](https://developercertificate.org/) (DCO).
By signing off a commit you certify that you wrote it, or otherwise have the
right to submit it under the project's license (Apache License 2.0).

Sign off with `-s`:

```bash
git commit -s -m "fix(route): handle empty host"
```

which adds a line like this to the commit message:

```
Signed-off-by: Your Name <you@example.com>
```

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).
