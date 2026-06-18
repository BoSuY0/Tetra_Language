#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

version=""
source_url=""
source_sha256=""
out_path=""

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/packages/render-homebrew-formula.sh \
  --version vX.Y.Z \
  --source-url URL \
  --source-sha256 SHA256 \
  --out PATH

Renders packaging/homebrew/Formula/tetra.rb.template into a Homebrew formula.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      shift
      version="${1:-}"
      ;;
    --source-url)
      shift
      source_url="${1:-}"
      ;;
    --source-sha256)
      shift
      source_sha256="${1:-}"
      ;;
    --out)
      shift
      out_path="${1:-}"
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "render-homebrew-formula: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ -z "$version" || -z "$source_url" || -z "$source_sha256" || -z "$out_path" ]]; then
  echo "render-homebrew-formula: --version, --source-url, --source-sha256, and --out are required" >&2
  usage >&2
  exit 2
fi
case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *)
    echo "render-homebrew-formula: version must look like vX.Y.Z, got $version" >&2
    exit 2
    ;;
esac
case "$source_sha256" in
  sha256:*) source_sha256="${source_sha256#sha256:}" ;;
esac
if [[ ! "$source_sha256" =~ ^[0-9a-fA-F]{64}$ ]]; then
  echo "render-homebrew-formula: invalid sha256 $source_sha256" >&2
  exit 2
fi

template_path="packaging/homebrew/Formula/tetra.rb.template"
formula="$(< "$template_path")"
formula="${formula//__VERSION__/$version}"
formula="${formula//__SOURCE_URL__/$source_url}"
formula="${formula//__SOURCE_SHA256__/$source_sha256}"

mkdir -p "$(dirname "$out_path")"
printf '%s\n' "$formula" > "$out_path"
echo "Rendered Homebrew formula: $out_path"
