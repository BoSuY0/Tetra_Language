# GitHub Issue Triage Implementation Plan

**Goal:** Make new GitHub issues self-classifying by type and project area.
**Context:** The repository currently has no `.github/ISSUE_TEMPLATE` files and only the default GitHub labels.
**Execution:** Implement directly in the current `main` checkout.

## Tasks

1. Add issue forms
   - **Files:** `.github/ISSUE_TEMPLATE/bug.yml`, `feature.yml`, `docs.yml`, `question.yml`, `config.yml`.
   - **Approach:** Provide four entry points with required summary/details fields and a shared `Area` dropdown.
   - **Verification:** Parse each YAML file and confirm required top-level keys exist.
   - **Done when:** The issue chooser offers bug, feature, docs, and question forms with explicit area choices.

2. Add automatic area labeling
   - **Files:** `.github/workflows/issue-triage.yml`.
   - **Approach:** On issue open/edit/reopen, parse the rendered issue body, read the `Area` section, remove old `area:*` labels, and add the matching one.
   - **Verification:** Run `actionlint` and YAML parsing.
   - **Done when:** The workflow can map every form area option to a repository label.

3. Create repository labels
   - **Labels:** `type:*`, `area:*`, and `status: needs triage`.
   - **Approach:** Use `gh label create` or `gh label edit` so issue forms and the workflow can apply labels immediately.
   - **Verification:** Query `gh label list` and confirm all expected labels exist.
   - **Done when:** All labels referenced by forms/workflow exist on GitHub.

4. Publish
   - **Approach:** Commit the GitHub issue triage configuration and push `main`.
   - **Verification:** Fetch `origin/main` and confirm files are present remotely.
   - **Done when:** `main` is clean and equal to `origin/main`.
