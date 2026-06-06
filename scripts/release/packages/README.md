# scripts/release/packages

Release packaging helpers for installable Tetra distributions.

- `build-release-archives.sh` builds the `linux-x64` binary archive, source
  archive, and checksums for GitHub Releases.
- `render-homebrew-formula.sh` renders a Homebrew formula from a release source
  archive URL and SHA256.

The GitHub Actions workflow `.github/workflows/release-packages.yml` wires these
helpers into GitHub Releases, GHCR image publishing, and optional Homebrew tap
updates.
