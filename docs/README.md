# Documentation layout

Where each document lives, and why it lives there.

| Location | Documents | Why there |
|---|---|---|
| Repository root | `README.md`, `LICENSE`, `CHANGELOG.md`, `AGENTS.md` | Read from the root by GitHub, pkg.go.dev, Go's license detection, the release workflow and agent tools. |
| Repository root | `CONTRIBUTING.md`, `GOVERNANCE.md`, `MAINTAINERS.md`, `ROADMAP.md`, `ADOPTERS.md`, `SECURITY.md` | Project and community files that visitors, CNCF reviewers and security tooling look for at the top level. |
| `.github/` | `CODE_OF_CONDUCT.md`, the pull request and issue templates | Files GitHub picks up from `.github/` and links from the repository's Community page. |
| `docs/` | [DEVELOPING.md](DEVELOPING.md), [architecture/](architecture) | How the code is organised and how to change it. |
| `docs/code/` | Generated package documentation | Written and committed by a workflow. Do not edit by hand or commit local changes to it. |
| `.claude/` | `CLAUDE.md`, `rules/`, `skills/`, `settings.json` | Claude Code project instructions, shared with every contributor who uses it. |

## Intention

The root is kept to what a tool or a first-time visitor needs to find there.
Everything else has one home by kind: policy in `.github/`, engineering
documentation in `docs/`.

`CODE_OF_CONDUCT.md` and `DEVELOPING.md` were at the root until October 2026
and moved for that reason. Links to their old paths are stale. `SECURITY.md`
moved with them and was brought back to the root: the security policy is
what a reporter and the OpenSSF checks look for first.

## Adding a document

- Explains how the code works or how to change it: `docs/`.
- A policy GitHub surfaces (security, conduct, support, funding): `.github/`.
- Add something to the root only if a tool reads it from there, and add it
  to the table above.
- Link a new document from the Community table in the
  [README](../README.md#community) or from [CONTRIBUTING.md](../CONTRIBUTING.md),
  so it can be found.
- Developer and contributor documentation stays in this repository; the
  documentation website covers the resources and their APIs for users.
