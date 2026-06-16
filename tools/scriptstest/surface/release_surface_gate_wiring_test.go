package surface

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tetra_language/tools/internal/gatecontract"
)

func TestReleaseSurfaceGateRunsAllSurfaceEvidenceSlices(t *testing.T) {
	root := repoRoot(t)
	gatePath := filepath.Join(root, "scripts", "release", "surface", "gate.sh")
	raw, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/gate.sh [--report-dir DIR]",
		"surface-headless-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-component-tree-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-real-window-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh --report-dir \"$report_dir\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-text-focus-input.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-component-tree.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-component-tree-api.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-minimal-toolkit.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-toolkit-reuse.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-headless-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-linux-x64-real-window-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_dir/surface-wasm32-web-browser-canvas-accessibility-metadata.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"",
		"go run ./tools/cmd/validate-artifact-hashes --manifest \"$report_dir/artifact-hashes.json\"",
		"tetra.surface.runtime.v1",
		"no legacy UI sidecar artifacts",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate script missing %q", want)
		}
	}
}

func TestReleaseSurfaceFinalReleaseGateRunsCurrentSurfaceV1Evidence(t *testing.T) {
	root := repoRoot(t)
	gatePath := filepath.Join(root, "scripts", "release", "surface", "release-gate.sh")
	raw, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatalf("read Surface final release gate script: %v", err)
	}
	gate := string(raw)

	contract := loadSurfaceReleaseContract(t, root)
	reportPaths := requiredReportPaths(contract)
	assertEqualOrderedStrings(t, parseBashStringArray(t, gate, "required_reports"), reportPaths, "release-gate required_reports")
	assertEqualOrderedStrings(t, ciArtifactPaths(t, contract), reportPaths, "contract ci_artifacts")

	assertSurfaceReleaseContractValidators(t, gate, contract,
		"validate-surface-runtime-release-summary",
		"validate-surface-security-report",
		"validate-surface-performance-budget",
		"validate-surface-dev-workflow",
		"validate-surface-inspector",
		"validate-surface-template-smoke",
		"validate-surface-reference-apps",
		"validate-surface-package",
		"validate-surface-crash-report",
		"validate-surface-i18n",
		"validate-surface-widget-migration",
		"validate-artifact-hashes",
		"validate-surface-release-state",
		"validate-surface-claims",
	)

	if contract.Producer != "scripts/release/surface/release-gate.sh" {
		t.Fatalf("Surface final release contract producer = %q", contract.Producer)
	}
	for _, want := range []string{
		"Usage: bash scripts/release/surface/release-gate.sh [--report-dir DIR]",
		`gate_contract="scripts/release/surface/contracts/surface-release-v1.json"`,
		`gate_contract_id="surface-release-v1"`,
		`if [[ "${TETRA_RUN_GATE_CONTRACT_EXEC:-}" != "1" ]]; then`,
		"go run ./tools/cmd/run-gate --contract \"$gate_contract\" --report-dir \"$report_dir_arg\"\n  exit $?",
		`if [[ "${TETRA_RUN_GATE_CONTRACT_ID:-}" != "$gate_contract_id" ]]; then`,
		`go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`,
		"source \"$script_dir/report-dir-guard.sh\"",
		"surface_release_require_fresh_report_dir \"$report_dir\" \"$repo_root\" \"surface_release_gate:\"",
		"\"producer\": \"scripts/release/surface/release-gate.sh\"",
		"Surface v1 release gate must fail, not skip, when Chromium-compatible browser, Linux Wayland/display, accessibility probe, or clipboard harness evidence is unavailable.",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface final release gate script missing %q", want)
		}
	}

	hashWrite := strings.Index(gate, "go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"")
	stateValidate := strings.Index(gate, "go run ./tools/cmd/validate-surface-release-state --report-dir \"$report_dir\" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json")
	if hashWrite < 0 || stateValidate < 0 {
		t.Fatalf("Surface final release gate must include artifact hash write and release-state validation")
	}
	if hashWrite > stateValidate {
		t.Fatalf("Surface final release gate must write artifact-hashes.json before validate-surface-release-state reads it")
	}
	delegationBlock := strings.Index(gate, "go run ./tools/cmd/run-gate --contract \"$gate_contract\" --report-dir \"$report_dir_arg\"\n  exit $?\nfi")
	guardedContractIDCheck := strings.Index(gate, `if [[ "${TETRA_RUN_GATE_CONTRACT_ID:-}" != "$gate_contract_id" ]]; then`)
	contractPreflight := strings.Index(gate, `go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`)
	freshReportGuard := strings.Index(gate, `surface_release_require_fresh_report_dir "$report_dir" "$repo_root" "surface_release_gate:"`)
	if delegationBlock < 0 || guardedContractIDCheck < 0 || contractPreflight < 0 || freshReportGuard < 0 {
		t.Fatalf("Surface final release gate must include non-dry-run delegation, guarded contract check, contract preflight, and fresh report-dir guard")
	}
	if strings.Count(gate, "--dry-run") != 1 {
		t.Fatalf("Surface final release gate must use dry-run exactly once inside guarded execution")
	}
	if strings.Count(gate, `go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg"`) != 2 {
		t.Fatalf("Surface final release gate must call run-gate once for non-dry-run delegation and once for guarded dry-run validation")
	}
	if contractPreflight < freshReportGuard {
		t.Fatalf("Surface final release gate must reject unsafe report directories before guarded contract preflight")
	}
	if contractPreflight < guardedContractIDCheck {
		t.Fatalf("Surface final release gate dry-run preflight must be inside guarded execution")
	}
	firstEvidenceStep := strings.Index(gate, `bash scripts/release/surface/surface-headless-release-smoke.sh --report-dir "$report_dir"`)
	if firstEvidenceStep < 0 {
		t.Fatalf("Surface final release gate must include the first headless release evidence step")
	}
	if delegationBlock > firstEvidenceStep {
		t.Fatalf("Surface final release gate must delegate normal execution before release evidence steps")
	}
	if contractPreflight > firstEvidenceStep {
		t.Fatalf("Surface final release gate must validate the contract inside guarded execution before release evidence steps")
	}
}

func TestReleaseSurfaceDocsClaimsGateRunsClaimScanner(t *testing.T) {
	root := repoRoot(t)
	gatePath := filepath.Join(root, "scripts", "release", "surface", "surface-docs-claims-gate.sh")
	raw, err := os.ReadFile(gatePath)
	if err != nil {
		t.Fatalf("read Surface docs claims gate script: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-docs-claims-gate.sh [--report-dir DIR]",
		`cmd=(go run ./tools/cmd/validate-surface-claims --root "$repo_root")`,
		`cmd+=(--report-dir "$report_dir")`,
		`"${cmd[@]}"`,
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface docs claims gate script missing %q", want)
		}
	}
}

func loadSurfaceReleaseContract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(root, "scripts", "release", "surface", "contracts", "surface-release-v1.json")
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load Surface release contract: %v", err)
	}
	return contract
}

func requiredReportPaths(contract gatecontract.Contract) []string {
	paths := make([]string, 0, len(contract.RequiredReports))
	for _, report := range contract.RequiredReports {
		paths = append(paths, report.Path)
	}
	return paths
}

func ciArtifactPaths(t *testing.T, contract gatecontract.Contract) []string {
	t.Helper()
	paths := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf("Surface release contract ci_artifacts entry %q must be required", artifact.Path)
		}
		paths = append(paths, artifact.Path)
	}
	return paths
}

func parseBashStringArray(t *testing.T, script, name string) []string {
	t.Helper()
	start := name + "=("
	inArray := false
	var values []string
	for lineNo, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inArray {
			if trimmed == start {
				inArray = true
			}
			continue
		}
		if trimmed == ")" {
			return values
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		value, err := strconv.Unquote(trimmed)
		if err != nil {
			t.Fatalf("parse %s entry on line %d: %v", name, lineNo+1, err)
		}
		values = append(values, value)
	}
	if !inArray {
		t.Fatalf("missing bash array %s", name)
	}
	t.Fatalf("bash array %s is not closed", name)
	return nil
}

func assertEqualOrderedStrings(t *testing.T, got, want []string, label string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d\ngot:  %q\nwant: %q", label, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q\ngot:  %q\nwant: %q", label, i, got[i], want[i], got, want)
		}
	}
}

func assertSurfaceReleaseContractValidators(t *testing.T, gate string, contract gatecontract.Contract, ids ...string) {
	t.Helper()
	validators := surfaceReleaseValidatorsByID(t, contract)
	for _, id := range ids {
		validator, ok := validators[id]
		if !ok {
			t.Fatalf("Surface release contract missing validator %q", id)
		}
		command := surfaceReleaseGateCommandFromContract(validator.Command)
		if !strings.Contains(gate, command) {
			t.Fatalf("Surface final release gate missing command for contract validator %q: %q", id, command)
		}
	}
}

func surfaceReleaseValidatorsByID(t *testing.T, contract gatecontract.Contract) map[string]gatecontract.Validator {
	t.Helper()
	validators := make(map[string]gatecontract.Validator, len(contract.Validators))
	for _, validator := range contract.Validators {
		if validator.ID == "" {
			t.Fatalf("Surface release contract contains validator with empty id")
		}
		if _, exists := validators[validator.ID]; exists {
			t.Fatalf("Surface release contract contains duplicate validator %q", validator.ID)
		}
		validators[validator.ID] = validator
	}
	return validators
}

func surfaceReleaseGateCommandFromContract(command string) string {
	command = strings.ReplaceAll(command, "$REPORT_DIR", "$report_dir")
	command = strings.ReplaceAll(command, "$REPO_ROOT", "$repo_root")
	return command
}

func assertOrderedFragments(t *testing.T, text string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		idx := strings.Index(text, fragment)
		if idx < 0 {
			t.Fatalf("missing ordered fragment %q", fragment)
		}
		if idx < last {
			t.Fatalf("fragment %q appears out of order", fragment)
		}
		last = idx
	}
}

func TestReleaseSurfaceTemplateSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-template-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface template smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-template-smoke.sh [--report-dir DIR]",
		"surface-template-smoke.json",
		"go run ./cli/cmd/tetra new surface-app",
		"--template command-palette",
		"--template settings",
		"--template dashboard",
		"--template editor-shell",
		"--template studio-shell",
		"--template multi-window-notes",
		"--template web-canvas",
		"go run ./cli/cmd/tetra check",
		"go run ./cli/cmd/tetra build --target linux-x64",
		"go run ./cli/cmd/tetra run --target linux-x64",
		"go run ./tools/cmd/surface-inspector",
		"go run ./tools/cmd/surface-visual-diff",
		"golden_runtime_dir=\"$runtime_dir/template-goldens\"",
		"--golden-artifact \"examples/surface_block_system.tetra,headless,1,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-1-initial.rgba\"",
		"--golden-artifact \"examples/surface_block_system.tetra,headless,2,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-2-focused.rgba\"",
		"--golden-artifact \"examples/surface_block_system.tetra,headless,3,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-3-motion.rgba\"",
		"tar -czf",
		"go run ./tools/cmd/validate-surface-template-smoke --report \"$report_path\"",
		"tetra.surface.template-smoke.v1",
		"surface-template-smoke-v1",
		"no React",
		"no Electron",
		"no DOM app UI tree",
		"no CSS runtime",
		"Block/Morph",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface template smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-template-smoke.sh --report-dir \"$report_dir\"",
		"surface-template-smoke.json",
		"\"project_templates\": \"surface-template-smoke-v1\"",
		"go run ./tools/cmd/validate-surface-template-smoke --report \"$report_dir/surface-template-smoke.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing template smoke wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-inspector-smoke.sh`,
		`surface-template-smoke.sh`,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-template-smoke --report "$report_dir/surface-template-smoke.json"`,
		`validate-surface-reference-apps --report "$report_dir/surface-reference-apps.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceReferenceAppsSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-reference-apps-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface reference apps smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-reference-apps-smoke.sh [--report-dir DIR]",
		"surface-reference-apps.json",
		"surface_reference_command_palette.tetra",
		"surface_reference_settings.tetra",
		"surface_reference_dashboard.tetra",
		"surface_reference_editor_shell.tetra",
		"surface_reference_file_manager.tetra",
		"surface_reference_dialog_notification.tetra",
		"surface_reference_localized_form.tetra",
		"surface_reference_accessibility_form.tetra",
		"surface_reference_multi_window_notes.tetra",
		"surface_reference_migration.tetra",
		"go run ./cli/cmd/tetra check \"$source\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$build_path\" \"$source\"",
		"go run ./cli/cmd/tetra run --target linux-x64 \"$source\"",
		"write_rgba_artifact",
		"\"artifact_path\": $(json_string \"$frame_path\")",
		"\"artifact_sha256\": $(json_string \"$frame_checksum\")",
		"\"golden_artifact_path\": $(json_string \"$golden_frame_path\")",
		"\"golden_artifact_sha256\": $(json_string \"$golden_frame_checksum\")",
		"\"self_golden_rejected\": true",
		"\"metadata_checksum_rejected\": true",
		"\"fixture_frame_only_rejected\": true",
		"\"missing_png_or_rgba_artifact_rejected\": true",
		"go run ./tools/cmd/validate-surface-reference-apps --report \"$report_path\"",
		"go run ./tools/cmd/validate-surface-visual-report --report \"$visual_report\"",
		"tetra.surface.reference-app-suite.v1",
		"surface-reference-app-suite-v1",
		"headless",
		"linux-x64-real-window",
		"wasm32-web-browser-canvas",
		"React",
		"Electron",
		"DOM app UI",
		"CSS runtime",
		"migration",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface reference apps smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-reference-apps-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-reference-apps.json",
		"\"reference_apps\": \"surface-reference-app-suite-v1\"",
		"go run ./tools/cmd/validate-surface-reference-apps --report \"$report_dir/surface-reference-apps.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing reference apps wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-template-smoke.sh`,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-reference-apps --report "$report_dir/surface-reference-apps.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfacePackageSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-package-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface package smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-package-smoke.sh [--report-dir DIR] [--source PATH] [--app-id ID] [--app-title TITLE] [--expected-exit-code N]",
		"surface-package.json",
		"surface_reference_command_palette.tetra",
		"surface_morph_rendered_studio_shell.tetra",
		"--expected-exit-code",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"go run ./cli/cmd/tetra build --target wasm32-web -o \"$wasm_binary\" \"$source_path\"",
		"linux_package_name=\"$app_binary_name-linux-x64\"",
		"web_package_name=\"$app_binary_name-wasm32-web\"",
		"$app_binary_name.mjs",
		"surface-browser-canvas-host.mjs",
		"scripts/tools/surface_browser_canvas_host.mjs",
		"runSurfaceBrowserCanvas",
		"id=\"surface-status\"",
		"new URL(\"./$app_binary_name.wasm\", import.meta.url)",
		"tetra.surface.package.v1",
		"surface-package-v1",
		"surface-app-package-v1",
		"tetra.surface.update-channel.v1",
		"hash-pinned-channel-manifest-v1",
		"auto_update_runtime_claim",
		"network_update_claim",
		"no_unsigned_signing_claim",
		"--expected-exit-code must be 0 for Surface package evidence",
		"preinstall_expected_exit_code=\"$expected_exit_code\"",
		"go run ./tools/cmd/validate-surface-package --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface package smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-package-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-package.json",
		"\"surface_package\": \"surface-package-v1\"",
		"go run ./tools/cmd/validate-surface-package --report \"$report_dir/surface-package.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing package wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-reference-apps-smoke.sh`,
		`surface-package-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestSurfaceBrowserCanvasHostKeepsStudioShellFullFrame(t *testing.T) {
	root := repoRoot(t)
	hostPath := filepath.Join(root, "scripts", "tools", "surface_browser_canvas_host.mjs")
	raw, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("read Surface browser canvas host: %v", err)
	}
	host := string(raw)
	for _, want := range []string{
		"function dispatchStudioShellBrowserInput(surface)",
		"scenario === 'studio-shell'",
		"clientX: 720",
		"clientY: 336",
	} {
		if !strings.Contains(host, want) {
			t.Fatalf("Surface browser canvas host missing studio-shell full-frame wiring %q", want)
		}
	}

	const marker = "function dispatchStudioShellBrowserInput(surface)"
	start := strings.Index(host, marker)
	if start < 0 {
		t.Fatalf("Surface browser canvas host missing %s", marker)
	}
	section := host[start:]
	if next := strings.Index(section[len(marker):], "\n  function "); next >= 0 {
		section = section[:len(marker)+next]
	}
	for _, forbidden := range []string{
		"canvas.width = 400",
		"canvas.height = 240",
		"dispatchCounterBrowserInput(surface)",
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("studio-shell browser input must not use cropped counter behavior %q:\n%s", forbidden, section)
		}
	}
}

func TestReleaseSurfaceI18nSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-i18n-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface i18n smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-i18n-smoke.sh [--report-dir DIR]",
		"surface-i18n.json",
		"surface_reference_localized_form.tetra",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"tetra.surface.i18n.v1",
		"surface-i18n-v1",
		"fallback_locale",
		"missing_key_diagnostic",
		"format_hooks",
		"rtl-placeholder-without-full-bidi-shaping-v1",
		"no_full_bidi_claim",
		"go run ./tools/cmd/validate-surface-i18n --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface i18n smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-i18n-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-i18n.json",
		"\"i18n_localization\": \"surface-i18n-v1\"",
		"go run ./tools/cmd/validate-surface-i18n --report \"$report_dir/surface-i18n.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing i18n wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-crash-report-smoke.sh`,
		`surface-i18n-smoke.sh`,
		`surface-widget-migration-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-crash-report --report "$report_dir/surface-crash-report.json"`,
		`validate-surface-i18n --report "$report_dir/surface-i18n.json"`,
		`validate-surface-widget-migration --report "$report_dir/surface-widget-migration.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceWidgetMigrationSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-widget-migration-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface widget migration smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-widget-migration-smoke.sh [--report-dir DIR]",
		"surface-widget-migration.json",
		"surface_reference_migration.tetra",
		"core_widgets_smoke.tetra",
		"go run ./cli/cmd/tetra check \"$source_path\"",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"tetra.surface.widget-migration.v1",
		"surface-widget-migration-v1",
		"lib.core.widgets",
		"Panel",
		"Button",
		"TextBox",
		"recipe_region_panel",
		"recipe_control_action",
		"recipe_field_text",
		"block_only_core_primitive",
		"no_future_core_primitive_promotion",
		"go run ./tools/cmd/validate-surface-widget-migration --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface widget migration smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-widget-migration-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-widget-migration.json",
		"\"widget_migration\": \"surface-widget-migration-v1\"",
		"go run ./tools/cmd/validate-surface-widget-migration --report \"$report_dir/surface-widget-migration.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing widget migration wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-i18n-smoke.sh`,
		`surface-widget-migration-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-i18n --report "$report_dir/surface-i18n.json"`,
		`validate-surface-widget-migration --report "$report_dir/surface-widget-migration.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceCrashReportSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-crash-report-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface crash report smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-crash-report-smoke.sh [--report-dir DIR]",
		"surface-crash-report.json",
		"surface_reference_command_palette.tetra",
		"go run ./cli/cmd/tetra build --target linux-x64 -o \"$linux_binary\" \"$source_path\"",
		"command_failure",
		"host_crash",
		"restart_recovery",
		"tetra.surface.diagnostic.v1",
		"surface-non-user-data-diagnostics-v1",
		"surface-diagnostic-redaction-v1",
		"scoped-linux-x64-process-restart-v1",
		"no_restart_claim_without_evidence",
		"no_electron_crash_reporter_dependency",
		"go run ./tools/cmd/validate-surface-crash-report --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface crash report smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-crash-report-smoke.sh --report-dir \"$report_dir_arg\"",
		"surface-crash-report.json",
		"\"crash_reporting\": \"surface-crash-report-v1\"",
		"go run ./tools/cmd/validate-surface-crash-report --report \"$report_dir/surface-crash-report.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing crash report wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-package-smoke.sh`,
		`surface-crash-report-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-package --report "$report_dir/surface-package.json"`,
		`validate-surface-crash-report --report "$report_dir/surface-crash-report.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceDevWorkflowSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-dev-workflow-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface dev workflow smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-dev-workflow-smoke.sh [--report-dir DIR]",
		"surface-dev-workflow.json",
		"go run ./cli/cmd/tetra surface dev",
		"--change-file \"token:$tokens_path\"",
		"--change-file \"recipe:$recipes_path\"",
		"--change-file \"source:$source_path\"",
		"go run ./tools/cmd/validate-surface-dev-workflow --report \"$report_path\"",
		"tetra.surface.dev-workflow.v1",
		"surface-dev-workflow-v1",
		"fast rebuild",
		"token/recipe/source",
		"no hot reload claim",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface dev workflow smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-dev-workflow-smoke.sh --report-dir \"$report_dir\"",
		"surface-dev-workflow.json",
		"\"developer_fast_loop\": \"surface-dev-workflow-v1\"",
		"go run ./tools/cmd/validate-surface-dev-workflow --report \"$report_dir/surface-dev-workflow.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing dev workflow wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-linux-x64-release-app-shell-smoke.sh`,
		`surface-dev-workflow-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-dev-workflow --report "$report_dir/surface-dev-workflow.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceInspectorSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-inspector-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface inspector smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-inspector-smoke.sh [--report-dir DIR]",
		"surface-inspector.json",
		"surface-inspector.html",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-morph",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-block-events",
		"go run ./tools/cmd/surface-inspector",
		"go run ./tools/cmd/validate-surface-inspector --report \"$report_path\"",
		"tetra.surface.inspector.v1",
		"surface-inspector-v1",
		"static tool report",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface inspector smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-inspector-smoke.sh --report-dir \"$report_dir\"",
		"surface-inspector.json",
		"\"inspector\": \"surface-inspector-v1\"",
		"go run ./tools/cmd/validate-surface-inspector --report \"$report_dir/surface-inspector.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing inspector wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-dev-workflow-smoke.sh`,
		`surface-inspector-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-inspector --report "$report_dir/surface-inspector.json"`,
		`validate-surface-release-state --report-dir "$report_dir"`,
	)
}

func TestReleaseSurfaceAppModelSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-headless-app-model-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface app-model smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-headless-app-model-smoke.sh [--report-dir DIR]",
		"surface-headless-app-model.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model",
		"--source examples/surface_app_model.tetra",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_path\" --release app-model",
		"tetra.surface.app-model.v1",
		"explicit command/reducer",
		"React hooks",
		"DOM event model",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface app-model smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-headless-app-model-smoke.sh --report-dir \"$report_dir\"",
		"surface-headless-app-model.json",
		"\"app_model\": \"explicit-command-reducer-v1\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing app-model wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-headless-release-accessibility-smoke.sh`,
		`surface-headless-app-model-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1`,
	)
}

func TestReleaseSurfaceLinuxAppShellSmokeAndGateWiring(t *testing.T) {
	root := repoRoot(t)
	scriptPath := filepath.Join(root, "scripts", "release", "surface", "surface-linux-x64-release-app-shell-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Surface linux app-shell smoke script: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh [--report-dir DIR]",
		"surface-linux-x64-release-app-shell.json",
		"go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell",
		"--source examples/surface_linux_app_shell_notes.tetra",
		"go run ./tools/cmd/validate-surface-runtime --report \"$report_path\" --release linux-app-shell",
		"go run ./tools/cmd/validate-surface-security-report --report \"$report_path\"",
		"go run ./tools/cmd/validate-surface-performance-budget --report \"$report_path\"",
		"tetra.surface.linux-app-shell.v1",
		"linux-app-shell-subset-v1",
		"electron feature ledger",
		"surface-security-permission-v1",
		"surface-performance-budget-v1",
		"startup/frame/memory/cache/framebuffer",
		"no faster-than-Electron claim",
		"capability-checked IPC/process boundaries",
		"local hashed asset/font/image",
		"multi-window notes",
		"lifecycle open/close/reopen",
		"resize/DPI/cursors",
		"file dialog",
		"file picker",
		"notification",
		"tray",
		"crash/error",
		"blocked-pass",
		"GTK/Qt/native widget UI",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Surface linux app-shell smoke script missing %q", want)
		}
	}

	gateRaw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate script: %v", err)
	}
	gate := string(gateRaw)
	for _, want := range []string{
		"surface-linux-x64-release-app-shell-smoke.sh --report-dir \"$report_dir\"",
		"surface-linux-x64-release-app-shell.json",
		"\"linux_app_shell\": \"linux-app-shell-subset-v1\"",
		"\"app_shell_features\": \"electron-feature-ledger-v1\"",
		"\"security_permissions\": \"surface-security-permission-v1\"",
		"\"performance_budget\": \"surface-performance-budget-v1\"",
		"go run ./tools/cmd/validate-surface-security-report --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
		"go run ./tools/cmd/validate-surface-performance-budget --report \"$report_dir/surface-linux-x64-release-app-shell.json\"",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing linux app-shell wiring %q", want)
		}
	}
	assertOrderedFragments(t, gate,
		`surface-linux-x64-release-window-smoke.sh`,
		`surface-linux-x64-release-app-shell-smoke.sh`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1`,
	)
}
