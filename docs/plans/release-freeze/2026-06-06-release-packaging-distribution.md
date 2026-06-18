# Release Packaging Distribution Implementation Plan

**Goal:** Add installable distribution paths for Tetra through GitHub Release assets, a GHCR image,
and a Homebrew tap workflow.

**Context:** The current public profile is `v0.4.0`, with `linux-x64` as the production baseline.
The repository already builds local CLI binaries through `scripts/dev/bootstrap.sh`, has GitHub
Actions CI, and exposes `tetra version`.

## Task 1: Release Archive Tooling

**Goal:** Produce a reproducible Linux x64 archive suitable for GitHub Releases.

**Files:** add `scripts/release/packages/build-release-archives.sh`; add `install.sh`; update
`README.md`.

**Approach:** Build `tetra` and `t` with `GOOS=linux GOARCH=amd64 CGO_ENABLED=0`, package them into
`tetra-<version>-linux-x64.tar.gz`, and emit `checksums.txt`. Keep the install script limited to
Linux x64 until other target evidence is promoted.

**Verification:** `bash -n`; run the package script locally; unpack the archive and run
`tetra version`; run `install.sh` against a local file URL.

**Done when:** The archive contains `bin/tetra`, `bin/t`, `README.md`, `LICENSE`, and the checksum
file records the archive hash.

## Task 2: GHCR Container Package

**Goal:** Publish an OCI image that fills GitHub Packages and lets users run the CLI without
installing it.

**Files:** add `Dockerfile`, `.dockerignore`, and `.github/workflows/release-packages.yml`.

**Approach:** Use a multi-stage Go build image to produce `/usr/local/bin/tetra` and
`/usr/local/bin/t`, then publish `ghcr.io/<owner>/tetra-language:<version>` and `latest` from the
release workflow.

**Verification:** `docker build` when Docker is available; `docker run ... tetra version` locally
when possible; `actionlint` for workflow syntax.

**Done when:** The workflow can authenticate with `GITHUB_TOKEN` and push GHCR images on tag or
manual dispatch.

## Task 3: Homebrew Tap Support

**Goal:** Make a Homebrew formula available for `BoSuY0/homebrew-tetra`.

**Files:** add `packaging/homebrew/Formula/tetra.rb.template`; add
`scripts/release/packages/render-homebrew-formula.sh`; wire optional tap update into the release
workflow.

**Approach:** Generate a formula from the release source archive URL and SHA256. The formula builds
from source with Go so it can support Homebrew without requiring platform-specific bottles
immediately. Optional workflow inputs can push the generated formula to the tap repository when
`HOMEBREW_TAP_TOKEN` is configured.

**Verification:** render the formula locally; run `ruby -c` if Ruby is available; verify workflow
syntax.

**Done when:** Release automation can publish the formula artifact and optionally commit it to the
tap repository.

## Task 4: Documentation and Evidence

**Goal:** Give users clear install commands and keep repository evidence fresh.

**Files:** update `README.md`; run `graphify update .` after code changes.

**Approach:** Add an install section covering curl installer, Docker/GHCR, and Homebrew. Keep
language scoped to `linux-x64` release assets and current `v0.4.0` truth.

**Verification:** `git diff --check`, shell syntax, local packaging checks, workflow lint where
available.

**Done when:** Docs match the implemented scripts and workflows without claiming unsupported target
runtime coverage.
