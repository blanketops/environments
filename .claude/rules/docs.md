---
paths:
  - "**/*.md"
---

# Markdown docs

- Prose and tables. No emoji and no new diagrams.
- Lead with what it is, then why, then how to begin. Cut what a reader
  cannot act on.
- Do not narrate "this repo"; give the information.
- Check every fact against the source before writing it: versions in
  `go.mod` and `git tag`, wiring in `resolution/contract_resolution.go`,
  paths against the tree. Do not carry claims over from the previous text.
- Describe a Kind as shipped or reconciled only if it is wired in. Code
  that exists but is not wired in is described as not wired in.
- When adding to a list or table, match the neighbouring entries.
- Do not edit `CHANGELOG.md` or anything under `docs/code/`; both are
  generated.
- Developer and contributor docs live in this repository, not the docs
  site.
- `docs/README.md` says where each document lives. Engineering docs go in
  `docs/`, GitHub policy files in `.github/`; add to the root only what a
  tool reads from there.
