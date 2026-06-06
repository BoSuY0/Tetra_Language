# GitHub Issue Triage Implementation Plan

**Goal:** Make new GitHub issues self-classifying by project area without requiring GitHub Actions.
**Context:** GitHub Actions is blocked by an account billing lock, so issue triage must work through static issue form labels.
**Execution:** Implement directly in the current `main` checkout.

## Tasks

1. Add issue forms
   - **Files:** `.github/ISSUE_TEMPLATE/compiler.yml`, `syntax.yml`, `memory.yml`, `runtime.yml`, `cli.yml`, `docs.yml`, `packages.yml`, `examples.yml`, `config.yml`.
   - **Approach:** Provide one entry point per project area. Each form applies its `area:*` label statically, without a workflow.
   - **Verification:** Parse each YAML file and confirm required top-level keys exist.
   - **Done when:** The issue chooser offers area-specific forms that apply area labels without Actions.

2. Avoid automatic Actions usage
   - **Files:** `.github/workflows/ci.yml`; remove `.github/workflows/issue-triage.yml`.
   - **Approach:** Keep CI jobs available for manual `workflow_dispatch`, but remove branch push, pull request, schedule, and issue triage workflow triggers.
   - **Verification:** Run `actionlint`, YAML parsing, and workflow tests.
   - **Done when:** Normal pushes no longer start GitHub-hosted runners.

3. Create repository labels
   - **Labels:** `type:*`, `area:*`, and `status: needs triage`.
   - **Approach:** Use `gh label create` or `gh label edit` so issue forms can apply labels immediately.
   - **Verification:** Query `gh label list` and confirm all expected labels exist.
   - **Done when:** All labels referenced by forms exist on GitHub.

4. Publish
   - **Approach:** Commit the GitHub issue triage configuration and push `main`.
   - **Verification:** Fetch `origin/main` and confirm files are present remotely.
   - **Done when:** `main` is clean and equal to `origin/main`.
