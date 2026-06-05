package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10WebSmokeWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "web-smoke.sh")
	assertLegacyFileRemoved(t, "scripts/release_v1_0_web_smoke.sh", "scripts/release/v1_0/web-smoke.sh directly")
	raw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned web smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/web-smoke.sh",
		`go run ./tools/cmd/validate-web-ui-smoke --report "$report_path"`,
		`status="blocked"`,
		`blocker="headless browser command failed: ${browser_runner}"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/release/v1_0/web-smoke.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, text, "scripts/release_v1_0_web_smoke.sh", "scripts/release/v1_0/web-smoke.sh")
}

func TestReleaseV10WebSmokeScriptValidatesReportBeforeExit(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-web-ui-smoke --report "$report_path"`,
		`status="blocked"`,
		`blocker="headless browser command failed: ${browser_runner}"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("web smoke script missing %q", want)
		}
	}
}

func TestReleaseV10WebSmokeScriptCapturesUISchemaEvidence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`mountTetraUI`,
		`ui_schema`,
		`ui_bundle_path`,
		`ui_module_path`,
		`UI smoke result missing ui=* metadata marker`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("web smoke script missing %q", want)
		}
	}
}

func TestReleaseV10WebSmokeScriptCapturesRuntimeSignals(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`runtime_trace`,
		`web_runtime_probe.tetra`,
		`stdout:ok`,
		`nonzero-exit:ok`,
		`failure-propagation:ok`,
		`repeated-instantiation:ok`,
		`ui-event-dispatch:web-command-dispatch`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("web smoke script missing runtime signal %q", want)
		}
	}
}

func TestReleaseV10WebSmokeScriptWritesUTCGeneratedAt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `date -u +%Y-%m-%dT%H:%M:%SZ`) {
		t.Fatalf("web smoke script must write UTC generated_at timestamps")
	}
	if strings.Contains(text, `%(%Y-%m-%dT%H:%M:%SZ)T`) {
		t.Fatalf("web smoke script must not format local time with a Z suffix")
	}
}

func TestReleaseV10WebSmokeScript_bindHTTPServerToLoopback(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	want := `python3 -m http.server "$port" --bind 127.0.0.1 --directory "$tmp_dir"`
	if !strings.Contains(text, want) {
		t.Fatalf("web smoke script must bind http.server to loopback, missing %q", want)
	}
}

func TestReleaseV10WebSmokeScriptUsesEphemeralPortAndTrapCleanup(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"))
	if err != nil {
		t.Fatalf("read web smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`cleanup_web_smoke()`,
		`trap cleanup_web_smoke EXIT`,
		`port="0"`,
		`wait_for_server_port "$tmp_dir/server.log"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("web smoke script missing ephemeral-port cleanup contract %q", want)
		}
	}
	if strings.Contains(text, "for candidate in 8711 8712 8713 8714 8715") {
		t.Fatalf("web smoke script must not use fixed local port range")
	}
}

func Test_release_v1_0_web_smokeRejectsMissingReportArgument(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)

	out, err := runReleaseV10WebSmoke(t, root, "--report")
	if err == nil {
		t.Fatalf("expected missing --report argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/web-smoke: --report requires a path") {
		t.Fatalf("missing report argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func Test_release_v1_0_web_smokeRejectsDirectoryReportBeforeSmokeOrBrowserSideEffects(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	reportPath := filepath.Join(root, "report-dir")
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected directory report path rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/web-smoke: refusing to use directory report path: "+reportPath) {
		t.Fatalf("directory report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("directory report path should block before build/browser side effects, stat err = %v", err)
	}
}

func Test_release_v1_0_web_smokeRejectsSymlinkReportBeforeSmokeOrBrowserSideEffects(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	targetPath := filepath.Join(root, "web-ui-smoke-target.json")
	if err := os.WriteFile(targetPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "web-ui-smoke-link.json")
	if err := os.Symlink(targetPath, reportPath); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected symlink report path rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/web-smoke: refusing to use directory report path: "+reportPath) {
		t.Fatalf("symlink report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("symlink report path should block before build/browser side effects, stat err = %v", err)
	}
}

func Test_release_v1_0_web_smokeAcceptsDashPrefixedReportPathAndSidecars(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	reportArg := "-web-ui-smoke.json"

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportArg)
	if err == nil {
		t.Fatalf("fallback UI source should still block after writing report\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if !strings.Contains(string(out), "fallback wasm web smoke ran successfully") {
		t.Fatalf("unexpected dash-prefixed web smoke output:\n%s", out)
	}
	for _, rel := range []string{reportArg, "-web-ui-smoke.dom.html", "-web-ui-smoke.chromium.err"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected dash-prefixed output %s: %v\n%s", rel, err, out)
		}
	}
}

func Test_release_v1_0_web_smokeDiscoversGoogleChromeFallback(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected fallback UI source to leave smoke blocked\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if report.Automation != "google-chrome --headless --no-sandbox --disable-gpu --disable-dev-shm-usage --disable-crash-reporter --disable-breakpad --dump-dom" {
		t.Fatalf("automation = %q, want google-chrome runner", report.Automation)
	}
	if !strings.Contains(report.Blocker, "fallback wasm web smoke ran successfully") {
		t.Fatalf("unexpected blocker: %q", report.Blocker)
	}
}

func Test_release_v1_0_web_smokeBrowserArgOverridesDiscovery(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "custom-chrome")
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--browser", "custom-chrome", "--report", reportPath)
	if err == nil {
		t.Fatalf("expected fallback UI source to leave smoke blocked\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Automation != "custom-chrome --headless --no-sandbox --disable-gpu --disable-dev-shm-usage --disable-crash-reporter --disable-breakpad --dump-dom" {
		t.Fatalf("automation = %q, want explicit custom-chrome runner", report.Automation)
	}
}

func Test_release_v1_0_web_smokeMissingExplicitBrowserWritesBlockedReport(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--browser", "missing-chrome", "--report", reportPath)
	if err == nil {
		t.Fatalf("expected missing explicit browser to block\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if report.Automation != "missing-chrome --headless --no-sandbox --disable-gpu --disable-dev-shm-usage --disable-crash-reporter --disable-breakpad --dump-dom" {
		t.Fatalf("automation = %q, want missing explicit browser in report", report.Automation)
	}
	if !strings.Contains(report.Blocker, "browser runner unavailable: missing-chrome") {
		t.Fatalf("unexpected blocker: %q", report.Blocker)
	}
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("missing browser should block before build, stat err = %v", err)
	}
}

func Test_release_v1_0_web_smokeRejectsMissingNodeBeforeBuild(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	if err := os.Remove(filepath.Join(root, "bin", "node")); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected missing node prerequisite to block\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !strings.Contains(report.Blocker, "node") {
		t.Fatalf("blocker = %q, want node prerequisite context", report.Blocker)
	}
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("missing node should block before build, stat err = %v", err)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func Test_release_v1_0_web_smokeRejectsMissingPythonBeforeBrowserPhase(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	writeReleaseV10WebSmokeUIBuildTetra(t, root)
	sourcePath := filepath.Join("examples", "ui_web_smoke.tetra")
	if err := os.WriteFile(filepath.Join(root, sourcePath), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "bin", "python3")); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--source", sourcePath, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected missing python3 prerequisite to block\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", report.Status)
	}
	if !strings.Contains(report.Blocker, "python3") || !strings.Contains(report.Blocker, "http.server") {
		t.Fatalf("blocker = %q, want python3 http.server prerequisite context", report.Blocker)
	}
	for _, suffix := range []string{".dom.html", ".chromium.err"} {
		if _, err := os.Stat(strings.TrimSuffix(reportPath, ".json") + suffix); !os.IsNotExist(err) {
			t.Fatalf("missing python3 should avoid browser sidecar %s, stat err = %v", suffix, err)
		}
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func Test_release_v1_0_web_smokeValidateWASMImportsFailureWritesStructuredFailReport(t *testing.T) {
	root := releaseV10WebSmokeFakeRepo(t)
	writeReleaseV10FakeBrowser(t, root, "google-chrome")
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(`#!/bin/sh
if [ "${1:-}" = "run" ] && [ "${2:-}" = "./tools/cmd/validate-wasm-imports" ]; then
  echo "validate-wasm-imports: disallowed import tetra_web_v1.fetch" >&2
  exit 1
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "report", "web-ui-smoke.json")

	out, err := runReleaseV10WebSmoke(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected validate-wasm-imports failure to fail smoke\n%s", out)
	}
	report := readWebSmokeReport(t, reportPath)
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if !strings.Contains(report.Blocker, "import validation") {
		t.Fatalf("blocker = %q, want import validation context", report.Blocker)
	}
}

func writeReleaseV10WebSmokeUIBuildTetra(t *testing.T, root string) {
	t.Helper()
	tetra := `#!/bin/sh
if [ "${1:-}" = "smoke" ]; then
  shift
  list_mode="false"
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --list)
        list_mode="true"
        shift
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ "$list_mode" = "true" ]; then
    printf '%s\n' '{"target":"wasm32-web","build_only":true,"run_supported":false,"total":1,"islands_debug":false,"cases":[{"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0}]}'
    exit 0
  fi
fi
if [ "${1:-}" = "build" ]; then
  out=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
      out="$2"
      shift 2
    else
      shift
    fi
  done
  if [ -z "$out" ]; then
    echo "missing -o" >&2
    exit 2
  fi
  printf '%s\n' "build" > tetra-build.log
  printf '%s\n' "export async function runTetra() { return 0; } export async function instantiateTetra() { return { instance: { exports: { tetra_main() {} } } }; }" > "$out.mjs"
  case "$out" in
    *web_smoke)
      printf '%s\n' '{"schema":"tetra.ui.v1","views":[]}' > "$out.ui.json"
      printf '%s\n' "export async function mountTetraUI() { return { schema: 'tetra.ui.v1', views: [] }; }" > "$out.ui.web.mjs"
      ;;
  esac
  exit 0
fi
echo "unexpected tetra command: $*" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
}
