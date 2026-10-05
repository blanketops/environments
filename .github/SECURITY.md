# Security Policy

This policy covers every repository of BlanketOps Environments, listed in
[GOVERNANCE.md](../GOVERNANCE.md#scope).

## Supported Versions

This project is pre-1.0. Security fixes are only made for the latest
released minor version — older releases do not receive patches.

| Version | Supported          |
| ------- | ------------------ |
| 0.8.x   | :white_check_mark: |
| < 0.8   | :x:                |

The other repositories version independently; for each, only the latest
release is supported.

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Report vulnerabilities privately by emailing **ntlaletsi86@gmail.com** with:

- a description of the issue and its impact;
- the affected repository and version (or commit);
- steps to reproduce, and any relevant logs or proof-of-concept code.

## What happens next

- You should receive an acknowledgement within 5 business days.
- We'll follow up with an assessment and, if accepted, a target timeline
  for a fix once the issue is confirmed.
- If declined, we'll explain why (e.g. out of scope, not reproducible, or
  intended behavior).
- We keep you informed while the fix is prepared, and credit you in the
  advisory unless you ask us not to.

## Disclosure

Please give us a reasonable window to ship a fix before any public
disclosure. Once a fixed release is available, we publish a GitHub security
advisory in the affected repository describing the issue, the affected
versions and the fix.

## Security practices

- Release assets are published with SLSA provenance.
- CodeQL, govulncheck and OpenSSF Scorecard run in CI.
- Native Go fuzz targets cover the parsing of contract data taken from
  Custom Resources.
