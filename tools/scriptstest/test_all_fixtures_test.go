package scriptstest

import (
	"os"
	"path/filepath"
	"testing"
)

func testAllFakeRepo(t *testing.T, failFmt bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v1_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "ci", "test-all.sh"), filepath.Join(root, "scripts", "ci", "test-all.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "dev", "bootstrap.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\ncp ./tetra ./t\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "ci", "test.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release", "v1_0", "wasi-smoke.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report) report=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p \"$(dirname \"$report\")\"\nprintf '{\"status\":\"pass\",\"cases\":[]}\\n' >\"$report\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release", "v1_0", "web-smoke.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report) report=\"$2\"; shift 2 ;; *) shift ;; esac; done\nif [[ \"${TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT:-}\" == \"1\" ]]; then exit 0; fi\nmkdir -p \"$(dirname \"$report\")\"\nprintf '{\"status\":\"pass\",\"ui_schema\":\"tetra.ui.bundle.v1\",\"cases\":[]}\\n' >\"$report\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release", "v1_0", "api-diff.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport_dir=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report-dir) report_dir=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p \"$report_dir\"\nprintf '{\"review\":{\"status\":\"clean\"},\"diff\":{\"added\":[],\"removed\":[],\"changed\":[]}}\\n' >\"$report_dir/api-diff.json\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "generated", "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goScript := `#!/usr/bin/env bash
set -euo pipefail
if [[ -n "${TETRA_FAKE_GO_LOG:-}" ]]; then
  printf '%s\n' "$*" >>"$TETRA_FAKE_GO_LOG"
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/validate-test-all-summary" && "${TETRA_FAIL_SUMMARY_VALIDATOR:-}" == "1" ]]; then
  echo "summary validator unavailable" >&2
  exit 23
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/gen-manifest" ]]; then
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    printf '{}\n' >"$out"
  fi
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/gen-docs" ]]; then
  printf '%s\n' '# Generated Tetra API Docs' ''
  printf '%s\n' '<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:ede46e5e34948c25f6ec38b0b963a2d8d42f5aa09071128581ee08271e966459","module_count":1,"entry_count":1} -->' ''
  printf '%s\n' '## examples' '' '### Functions' ''
  printf '%b\n' '- \x60func main() -> Int\x60'
  exit 0
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/validate-safety-readiness" ]]; then
  if [[ "${TETRA_FAIL_SAFETY_READINESS:-}" == "1" ]]; then
    echo "safety readiness failed" >&2
    exit 19
  fi
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --out)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    mkdir -p "$(dirname "$out")"
    printf '{"schema":"tetra.safety-readiness.v1","status":"pass","version":"v0.4.0","required_features":[]}\n' >"$out"
  fi
  exit 0
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
	case "$cmd" in
	  version)
	    echo "${TETRA_FAKE_TETRA_VERSION:-v0.4.0}"
	    ;;
  fmt)
    if [[ "${TETRA_FAIL_FMT:-}" == "1" ]]; then
      echo "format mismatch" >&2
      exit 7
    fi
    ;;
  test)
    for arg in "$@"; do
      if [[ "$arg" == "--report=json" ]]; then
        echo '{"total":0,"passed":0,"failed":0,"files":[],"results":[]}'
        exit 0
      fi
    done
    ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*) echo '{"code":"TETRA2001","message":"function main uses effect '\''io'\'' but does not declare it","severity":"error"}' >&2 ;;
          *tabs-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"tabs are not supported in Flow indentation","severity":"error"}' >&2 ;;
          *planned-actor-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"actor declarations currently support state fields and func methods only","severity":"error"}' >&2 ;;
          *) echo '{"code":"TETRA2001","message":"unknown function missing_call","severity":"error"}' >&2 ;;
        esac
        exit 1
      fi
    done
    ;;
  doc)
    printf '%s\n' '# Tetra API Docs' ''
    printf '%s\n' '<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:ede46e5e34948c25f6ec38b0b963a2d8d42f5aa09071128581ee08271e966459","module_count":1,"entry_count":1} -->' ''
    printf '%s\n' '## examples' '' '### Functions' ''
    printf '%b\n' '- \x60func main() -> Int\x60'
    ;;
  build)
    out=""
    prev=""
    for arg in "$@"; do
      if [[ "$prev" == "-o" ]]; then
        out="$arg"
      fi
      prev="$arg"
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      printf '\x00\x61\x73\x6d\x01\x00\x00\x00' >"$out"
    fi
    ;;
  targets)
    printf '{"supported":["linux-x64","windows-x64","macos-x64"],"build_only":["wasm32-wasi","wasm32-web"],"planned":[]}\n'
    ;;
  features)
    printf '{"schema":"tetra.features.v1","version":"v0.4.0","features":[]}\n'
    ;;
  doctor)
    if [[ "${TETRA_FAKE_ZERO_DOCTOR_REPORT:-}" == "1" ]]; then
      exit 0
    fi
    printf '{"status":"pass","checks":[{"name":"version","status":"pass"},{"name":"supported targets","status":"pass"},{"name":"build-only targets","status":"pass"},{"name":"planned targets","status":"pass"},{"name":"repo root","status":"pass"},{"name":"__rt/actors_sysv.tetra","status":"pass"},{"name":"__rt/actors_win64.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},{"name":"examples/flow_hello.tetra","status":"pass"},{"name":"docs/generated/manifest.json","status":"pass"},{"name":"docs manifest version","status":"pass"},{"name":"docs manifest surface","status":"pass"},{"name":"smoke sources","status":"pass"},{"name":"runtime exports","status":"pass"}]}\n'
    ;;
  smoke)
    report=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --report)
          report="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$report" ]]; then
      printf '{"target":"linux-x64","cases":[]}\n' >"$report"
    fi
    echo "Smoke linux-x64: 0/0 passed"
    ;;
  lsp)
    if [[ "${1:-}" == "--stdio" ]]; then
      cat >/dev/null
      printf '{"result":{"capabilities":{}}}\n'
      printf '{"method":"textDocument/publishDiagnostics","params":{"diagnostics":[]}}\n'
    else
      printf '{"diagnostics":[]}\n'
    fi
    ;;
  eco)
    sub="${1:-}"
    shift || true
    case "$sub" in
      verify)
        lock=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --lock)
              lock="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$lock" ]]; then
          mkdir -p "$(dirname "$lock")"
          cat >"$lock" <<'JSON'
{
  "schema": "tetra.eco.lock.v1",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "graph_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}
JSON
        fi
        ;;
      seed)
        action="${1:-}"
        shift || true
        case "$action" in
          export)
            out="tetra.seed.json"
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --out)
                  out="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            mkdir -p "$(dirname "$out")"
            printf '{}\n' >"$out"
            ;;
          import)
            lock=""
            capsules_dir=""
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --lock)
                  lock="$2"
                  shift 2
                  ;;
                --capsules-dir)
                  capsules_dir="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            if [[ -n "$lock" ]]; then
              mkdir -p "$(dirname "$lock")"
              printf '{}\n' >"$lock"
            fi
            if [[ -n "$capsules_dir" ]]; then
              mkdir -p "$capsules_dir"
              printf 'capsule App:\n  id "tetra://app"\n  version "0.1.0"\n' >"$capsules_dir/App.capsule"
            fi
            ;;
        esac
        ;;
      needmap)
        out="tetra.needmap.json"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$(dirname "$out")"
        printf '{}\n' >"$out"
        ;;
      pack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o|--out)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      unpack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$out/src"
          printf 'capsule App:\n  id "tetra://app"\n  version "0.1.0"\n  target "linux-x64"\n' >"$out/Tetra.capsule"
          printf 'func main() -> Int:\n    return 0\n' >"$out/src/main.tetra"
          printf '{"schema":"tetra.eco.package.v1","compression":"gzip","mtime_unix":0,"file_count":2,"files":[{"path":"Tetra.capsule","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":68},{"path":"src/main.tetra","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":32}]}\n' >"$out/tetra.package.json"
        fi
        ;;
      vault)
        action="${1:-}"
        shift || true
        store=".tetra/todex-vault"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --store)
              store="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$store/objects/sha256"
        printf '{}' >"$store/records.json"
        if [[ "$action" == "add" ]]; then
          echo "Vault added: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa source fixture"
        fi
        if [[ "$action" == "verify" ]]; then
          echo "Vault OK: 1 records"
        fi
        ;;
      trust)
        action="${1:-}"
        shift || true
        if [[ "$action" == "snapshot" ]]; then
          out="tetra.trust-snapshot.json"
          while [[ $# -gt 0 ]]; do
            case "$1" in
              -o)
                out="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          mkdir -p "$(dirname "$out")"
          printf '{}\n' >"$out"
        fi
        ;;
      materialize)
        out="."
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$out"
        printf '{}\n' >"$out/tetra.materialization.json"
        ;;
      publish)
        registry=".tetra/registry-beta"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --registry)
              registry="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$registry/packages/tetra_app/0.1.0/linux-x64"
        printf 'todex\n' >"$registry/packages/tetra_app/0.1.0/linux-x64/package.todex"
        printf '{"schema":"tetra.eco.publish.v1beta","channel":"beta","hub":"local-beta","published_at_unix":0,"capsule":{"id":"tetra://app","name":"App","version":"0.1.0","target":"linux-x64","targets":["linux-x64"],"permissions":["io"]},"package":{"file":"package.todex","size":6,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"downloads":[{"target":"linux-x64","path":"packages/tetra_app/0.1.0/linux-x64/package.todex"}]}\n' >"$registry/packages/tetra_app/0.1.0/linux-x64/metadata.json"
        ;;
      download)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      tetrahub)
        action="${1:-}"
        shift || true
        if [[ "$action" == "publish" ]]; then
          store=".tetra/tetrahub-beta"
          while [[ $# -gt 0 ]]; do
            case "$1" in
              --store)
                store="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          mkdir -p "$store/packages/tetra_app/0.1.0/linux-x64"
          printf 'todex\n' >"$store/packages/tetra_app/0.1.0/linux-x64/package.todex"
        elif [[ "$action" == "download" ]]; then
          out=""
          while [[ $# -gt 0 ]]; do
            case "$1" in
              -o)
                out="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          if [[ -n "$out" ]]; then
            mkdir -p "$(dirname "$out")"
            printf 'todex\n' >"$out"
          fi
        fi
        ;;
    esac
    ;;
  *)
    ;;
esac
`
	if failFmt && len(tetra) == 0 {
		t.Fatal("unreachable")
	}
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
