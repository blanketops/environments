---
paths:
  - ".github/workflows/**"
---

# Workflows

- Start the file with the comment header the other workflows use: what it
  does, its triggers, and the secrets it requires.
- Use the same `runs-on` as the existing workflows.
- Top-level `permissions` is `contents: read`. Grant anything more on the
  job that needs it, with a comment saying why.
- Authenticate to other `blanketops` repositories with the GitHub App token
  (`actions/create-github-app-token`, `APP_ID` / `APP_PRIVATE_KEY`), listing
  only the repositories the job needs. Do not add new uses of `GH_PAT`.
- A job that uses secrets must not run for pull requests from forks.
- End every job with a step that writes a summary to
  `$GITHUB_STEP_SUMMARY`, with `if: always()`.
- Run `actionlint` on the file before committing.
- Say in the pull request which steps have and have not run on a real
  runner.
