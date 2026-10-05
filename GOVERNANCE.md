# Governance

This document describes how BlanketOps Environments is run: who makes
decisions, how, and how to become part of that.

## Scope

BlanketOps Environments is developed across these repositories, all under the
Apache License 2.0. This governance applies to all of them:

| Repository | Purpose |
|---|---|
| [environments](https://github.com/blanketops/environments) | Resolution engine — turns Custom Resources into execution plans (this repository) |
| [environments-api](https://github.com/blanketops/environments-api) | Kubernetes API types and CRD definitions |
| [environments-contract](https://github.com/blanketops/environments-contract) | Canonical contracts shared by every component |
| [environments-controller](https://github.com/blanketops/environments-controller) | Kubernetes controller that runs the reconciliation loops |
| [environments-cli](https://github.com/blanketops/environments-cli) | Command-line client |
| [environments-install](https://github.com/blanketops/environments-install) | Declarative installation (CRDs and controller manifests) |
| [environments-tests](https://github.com/blanketops/environments-tests) | Conformance test suite for the API surface |
| [environments-docs](https://github.com/blanketops/environments-docs) | Documentation website |
| [secure-software-supplychain](https://github.com/blanketops/secure-software-supplychain) | Supply Chain plugin (Tekton, Kaniko, Trivy, Cosign, Grafeas) |

## Roles

### Contributors

Anyone who opens an issue, reviews a pull request, improves the docs or
contributes code is a contributor. See [CONTRIBUTING.md](CONTRIBUTING.md).

### Maintainers

Maintainers are listed in [MAINTAINERS.md](MAINTAINERS.md). They:

- review and merge pull requests;
- triage issues and set priorities;
- cut releases;
- own the [roadmap](ROADMAP.md);
- handle security reports ([SECURITY.md](.github/SECURITY.md)) and Code of Conduct
  reports ([CODE_OF_CONDUCT.md](.github/CODE_OF_CONDUCT.md)).

Maintainers act in the interest of the project and its users, not of any
employer.

## Decision making

Day-to-day decisions are made in the open, on GitHub issues and pull
requests, by **lazy consensus**: a proposal goes ahead if no maintainer
objects within a reasonable time — at least three working days for anything
beyond routine changes.

Changes that need explicit agreement:

| Change | Needs |
|---|---|
| A breaking change to a published API or contract | Approval from a majority of maintainers, recorded on the pull request |
| Adding or removing a maintainer | A majority of the other maintainers |
| Changing this governance document | Two-thirds of maintainers |

If consensus cannot be reached, maintainers vote, and a simple majority
decides. While there is a single maintainer, that maintainer decides, and
records the reasoning on the relevant issue or pull request.

## Becoming a maintainer

A contributor can be nominated as a maintainer — by themselves or by an
existing maintainer — after a sustained record of high-quality contributions
and reviews, and showing judgement that matches the project's
[design principles](README.md#design-principles). The nomination is an issue
in this repository; it is decided as described above.

Maintainers who are inactive for six months, or who ask to step down, move to
an emeritus list in [MAINTAINERS.md](MAINTAINERS.md).

The project currently has a single maintainer. Growing the maintainer group,
including maintainers from outside BlanketOps, is a priority, so that no
single person or company controls the project.

## Relationship with BlanketOps

BlanketOps Environments is started and currently maintained by people at
BlanketOps, which also builds commercial products — for example
organization, tenancy, networking and policy management — on top of it.

To keep the project independent of those products:

- Everything BlanketOps Environments does lives in the repositories listed
  under [Scope](#scope), under the Apache License 2.0. No feature of the
  project is held back for, or only available in, a commercial product.
- Commercial products are separate codebases. They use BlanketOps
  Environments through its public APIs, like any other adopter, and are not
  part of this project.
- Decisions about this project follow this document, in public, whatever
  BlanketOps' commercial plans are.

## Code of Conduct

Everyone in the project follows the [Code of Conduct](.github/CODE_OF_CONDUCT.md).

## Changes to this document

Changes are made by pull request to this repository, approved as described
under [Decision making](#decision-making).
