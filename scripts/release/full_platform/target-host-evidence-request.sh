#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
out_dir="$repo_root/reports/full-platform-ui-runtime/target-host-request"
repo=""
branch=""
remote_url=""

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/full_platform/target-host-evidence-request.sh [--out-dir DIR] [--repo OWNER/REPO] [--branch BRANCH]

Writes a target-host evidence request bundle for real Windows/macOS UI runtime
smoke collection. This bundle is not runtime evidence and never replaces the
required target-host reports.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --out-dir requires a value" >&2
        usage >&2
        exit 2
      fi
      out_dir="$2"
      shift 2
      ;;
    --repo)
      if [[ $# -lt 2 ]]; then
        echo "error: --repo requires a value" >&2
        usage >&2
        exit 2
      fi
      repo="$2"
      shift 2
      ;;
    --branch)
      if [[ $# -lt 2 ]]; then
        echo "error: --branch requires a value" >&2
        usage >&2
        exit 2
      fi
      branch="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"

if [[ -z "$repo" ]]; then
  remote_url="$(git remote get-url origin 2> /dev/null || true)"
  repo="$(printf '%s' "$remote_url" | sed -E 's#^git@github.com:##; s#^https://github.com/##; s#\.git$##')"
fi
if [[ -z "$branch" ]]; then
  branch="$(git rev-parse --abbrev-ref HEAD)"
fi
if [[ -z "$repo" || "$repo" == "$remote_url" ]]; then
  echo "error: could not infer GitHub repo; pass --repo OWNER/REPO" >&2
  exit 2
fi
if ! command -v jq > /dev/null 2>&1; then
  echo "error: jq is required for target-host evidence request generation" >&2
  exit 2
fi

mkdir -p "$out_dir"

version="$("./tetra" version 2> /dev/null || go run ./cli/cmd/tetra version)"
git_head="$(git rev-parse HEAD)"
request_json="$out_dir/target-host-evidence-request.json"
request_readme="$out_dir/README.md"

windows_command="git clone https://github.com/$repo.git tetra-ui-runtime && cd tetra-ui-runtime && git fetch origin $branch && git checkout $git_head && pwsh -File scripts/release/full_platform/windows-ui-runtime-smoke.ps1 -Report windows-ui-runtime.json -ExpectedVersion $version -ExpectedGitHead $git_head"
macos_command="git clone https://github.com/$repo.git tetra-ui-runtime && cd tetra-ui-runtime && git fetch origin $branch && git checkout $git_head && bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh --target macos-x64 --report macos-ui-runtime.json --expected-version $version --expected-git-head $git_head"
aggregation_command="TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime"

jq -n \
  --arg repo "$repo" \
  --arg branch "$branch" \
  --arg version "$version" \
  --arg gitHead "$git_head" \
  --arg windowsCommand "$windows_command" \
  --arg macosCommand "$macos_command" \
  --arg aggregationCommand "$aggregation_command" \
  '{
    schema: "tetra.ui.target-host-evidence-request.v1",
    status: "request",
    production_evidence: false,
    repo: $repo,
    branch: $branch,
    expected_version: $version,
    expected_git_head: $gitHead,
    warning: "This request bundle is not runtime evidence. Only validator-passing target-host reports from the same Git commit count.",
    targets: [
      {
        target: "windows-x64",
        host_requirement: "real Windows x64 host",
        report: "windows-ui-runtime.json",
        command: $windowsCommand
      },
      {
        target: "macos-x64",
        host_requirement: "real macOS x64 host",
        report: "macos-ui-runtime.json",
        command: $macosCommand
      }
    ],
    aggregation: {
      host_requirement: "Linux aggregation host with the same Git commit checked out",
      command: $aggregationCommand
    }
  }' > "$request_json"

cat > "$request_readme" << EOF
# Full-Platform UI Runtime Target-Host Evidence Request

This bundle is not runtime evidence. It records the exact commit and commands
needed to collect real Windows/macOS target-host UI runtime reports.

- Repository: $repo
- Branch: $branch
- Expected version: $version
- Expected Git HEAD: $git_head

Run each target on the same Git commit:

## Windows x64

\`\`\`powershell
$windows_command
\`\`\`

## macOS x64

\`\`\`bash
$macos_command
\`\`\`

Copy \`windows-ui-runtime.json\` and \`macos-ui-runtime.json\` to the Linux
aggregation host, then run:

\`\`\`bash
$aggregation_command
\`\`\`

Only reports that pass \`validate-windows-ui-runtime\`,
\`validate-macos-ui-runtime\`, and the cross-platform gate count as production
evidence. Build-only, metadata-only, runtime-less, fake, placeholder, or stale
reports remain blockers.
EOF

go run ./tools/cmd/validate-target-host-evidence-request \
  --report "$request_json" \
  --expected-repo "$repo" \
  --expected-branch "$branch" \
  --expected-version "$version" \
  --expected-git-head "$git_head"

echo "target-host evidence request bundle: $out_dir"
