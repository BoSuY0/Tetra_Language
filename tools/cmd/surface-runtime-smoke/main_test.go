package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func componentAppArtifacts(path string) []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{Kind: "component-app", Path: path, SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
	}
}

func headlessSurfaceArtifacts(path string) []surface.ArtifactReport {
	artifacts := componentAppArtifacts(path)
	return append(artifacts, surface.ArtifactReport{
		Kind:   "runner-trace",
		Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
		SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Size:   409,
	})
}

func wasmSurfaceArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-counter.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 7502},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-counter.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4931},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 413},
	}
}

func wasmBrowserCanvasSurfaceArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-browser-counter.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-browser-counter.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
	}
}

func cleanArtifactScan(filesChecked int) surface.ArtifactScanReport {
	return surface.ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: filesChecked, ForbiddenPaths: []string{}, Pass: true}
}

func TestSurfaceComponentAppExpectedExitForGeneratedTemplateSource(t *testing.T) {
	source := "reports/surface-electron-react-beauty-production/P21/template-smoke/templates/command-palette/src/main.tetra"
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", source); got != 0 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(generated template) = %d, want 0", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-morph", "examples/surface_reference_command_palette.tetra"); got != 0 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(reference app) = %d, want 0", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-morph", filepath.Join("/tmp", "repo", "examples", "surface_reference_settings.tetra")); got != 0 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(absolute reference app) = %d, want 0", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", "examples/surface_migration_tetra_control_center.tetra"); got != 5 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(flagship Control Center) = %d, want 5", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", filepath.Join("/tmp", "repo", "examples", "surface_migration_tetra_control_center.tetra")); got != 5 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(absolute flagship Control Center) = %d, want 5", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", "examples/surface_block_system.tetra"); got != 1 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(canonical block system) = %d, want 1", got)
	}
}

func TestSurfaceTemplateScenarioRetargetsOnlyGeneratedSources(t *testing.T) {
	generated := smokeOptions{Mode: "headless-block-system", SourcePath: "reports/surface-electron-react-beauty-production/P21/template-smoke/templates/command-palette/src/main.tetra"}
	if !shouldRetargetSurfaceTemplateScenario(generated) {
		t.Fatalf("generated template source should retarget block-system scenario")
	}
	canonical := smokeOptions{Mode: "headless-block-system", SourcePath: "examples/surface_block_system.tetra"}
	if shouldRetargetSurfaceTemplateScenario(canonical) {
		t.Fatalf("canonical block-system source must not retarget to generated template module")
	}
	counter := smokeOptions{Mode: "headless", SourcePath: "reports/surface-electron-react-beauty-production/P21/template-smoke/templates/command-palette/src/main.tetra"}
	if shouldRetargetSurfaceTemplateScenario(counter) {
		t.Fatalf("non Block/Morph mode must not retarget generated template scenario")
	}
}

func TestSurfaceTemplateSmokeUsesCanonicalRuntimeSources(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "surface-template-smoke.sh"))
	if err != nil {
		t.Fatalf("read surface-template-smoke.sh: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source examples/surface_block_system.tetra --report "$block_report"`,
		`go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source examples/surface_morph_command_palette.tetra --report "$morph_report"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("surface-template-smoke.sh missing canonical runtime source command %q", want)
		}
	}
	for _, forbidden := range []string{
		`--mode headless-block-system --source "$first_source"`,
		`--mode headless-morph --source "$first_source"`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("surface-template-smoke.sh must not run template source through synthetic runtime evidence: found %q", forbidden)
		}
	}
}

func TestMorphRenderedFlagshipSourcePresentsSurfaceFrames(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "examples", "surface_morph_rendered_studio_shell.tetra"))
	if err != nil {
		t.Fatalf("read Morph rendered flagship source: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`surface.open("Tetra Studio Shell", 320, 200)`,
		"surface.begin_frame(win)",
		"morph.render_studio_shell_frame(false, before_frame)",
		"surface.present(before_frame)",
		"surface.poll_event(win)",
		"surface.poll_event_text_into(win, text_bytes)",
		"win.width = resize_event.width",
		"morph.render_studio_shell_frame(active, after_frame)",
		"surface.present(after_frame)",
		"surface.close(win)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Morph rendered flagship source missing target runtime evidence marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"import lib.core.draw",
		"draw_flagship_shell_scene",
		"draw.DrawContext",
		"draw.rect",
		"draw.clear",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Morph rendered flagship source must stay Morph-authored; found forbidden marker %q", forbidden)
		}
	}
}

func TestHeadlessCounterScenarioProducesValidSurfaceRuntimeEvidence(t *testing.T) {
	scenario := runHeadlessCounterScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless",
		SourcePath: "examples/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-counter", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-counter"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "deterministic-headless" || report.HostEvidence.Backend != "software-rgba" || !report.HostEvidence.Framebuffer || report.HostEvidence.RealWindow || report.HostEvidence.NativeInput {
		t.Fatalf("host evidence = %#v, want deterministic headless software framebuffer without real-window/native-input claim", report.HostEvidence)
	}
	if !strings.Contains(string(raw), `"key":0`) || !strings.Contains(string(raw), `"timestamp_ms":0`) {
		t.Fatalf("report JSON %s, want explicit zero-valued key and timestamp_ms event fields", raw)
	}
	if len(report.Artifacts) != 2 || report.Artifacts[0].Kind != "component-app" || report.Artifacts[0].Path != "/tmp/surface-artifacts/surface-counter" || report.Artifacts[0].SHA256 == "" || report.Artifacts[1].Kind != "runner-trace" {
		t.Fatalf("artifacts = %#v, want component app and runner trace hash evidence", report.Artifacts)
	}
	if len(scenario.Frames) != 2 || scenario.Frames[0].Checksum == "" || scenario.Frames[1].Checksum == "" {
		t.Fatalf("scenario pre/post frame checksums missing: %#v", scenario.Frames)
	}
	if scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("scenario pre/post frame checksums match, want redraw evidence across state change: %#v", scenario.Frames)
	}
	if len(scenario.StateTransitions) != 2 || scenario.StateTransitions[0].Before != "0" || scenario.StateTransitions[0].After != "1" || scenario.StateTransitions[1].Field != "text_count" {
		t.Fatalf("state transitions = %#v, want count 0->1 and text_count 0->1", scenario.StateTransitions)
	}
	if len(scenario.Components) != 2 || !componentAbilitiesContainAll(scenario.Components[0].Abilities, []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}) || !componentAbilitiesContainAll(scenario.Components[1].Abilities, []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}) {
		t.Fatalf("component abilities = %#v, want ordinary struct measure/layout/draw/event/focus/text/accessibility evidence", scenario.Components)
	}
	if scenario.Components[1].ID != "CounterButton" {
		t.Fatalf("component hierarchy = %#v, want CounterButton child component evidence", scenario.Components)
	}
	if scenario.Components[0].Bounds.W != 320 || scenario.Components[0].Bounds.H != 200 || scenario.Components[1].Bounds.X != 32 || scenario.Components[1].Bounds.Y != 80 || scenario.Components[1].Bounds.W != 160 || scenario.Components[1].Bounds.H != 48 {
		t.Fatalf("component bounds = %#v, want measured/layout child bounds evidence", scenario.Components)
	}
	if len(scenario.Events) < 2 || scenario.Events[1].TargetComponent != "CounterButton" || scenario.Events[1].Width != 320 || scenario.Events[1].Height != 200 || !intSlicesEqual(scenario.Events[1].BufferSlots, []int{5, 48, 96, 1, 0, 320, 200, 0, 0}) {
		t.Fatalf("events = %#v, want full host event buffer dispatched to child CounterButton", scenario.Events)
	}
	if len(scenario.Events) < 2 || !stringSlicesEqual(scenario.Events[1].DispatchPath, []string{"CounterApp", "CounterButton"}) {
		t.Fatalf("events = %#v, want root-to-child dispatch path evidence", scenario.Events)
	}
	if len(scenario.Events) < 3 || scenario.Events[2].Kind != "text_input" || scenario.Events[2].TargetComponent != "CounterButton" || scenario.Events[2].TextLen != 2 || scenario.Events[2].TextBytesHex != "4f4b" {
		t.Fatalf("events = %#v, want host text payload dispatched to child CounterButton", scenario.Events)
	}
	if !intSlicesEqual(scenario.Events[2].BufferSlots, []int{8, 0, 0, 0, 0, 320, 200, 1, 2}) {
		t.Fatalf("events = %#v, want full host text event buffer dispatched to child CounterButton", scenario.Events)
	}
	if !caseNamesContain(scenario.Cases, "component hierarchy dispatch") {
		t.Fatalf("scenario cases = %#v, want static component hierarchy dispatch evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "host event buffer poll_event") {
		t.Fatalf("scenario cases = %#v, want host event buffer evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "component text input scalar dispatch") {
		t.Fatalf("scenario cases = %#v, want static component text input scalar evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "host text payload buffer") {
		t.Fatalf("scenario cases = %#v, want host text payload buffer evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "component focus dispatch") {
		t.Fatalf("scenario cases = %#v, want static component focus dispatch evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "component accessibility metadata") {
		t.Fatalf("scenario cases = %#v, want static component accessibility metadata evidence", scenario.Cases)
	}
}

func TestCollectHeadlessProcessEvidenceRecordsRunnerTrace(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	reportPath := filepath.Join(t.TempDir(), "surface-headless.json")
	source, err := resolveSurfaceSourcePath("examples/surface_counter.tetra")
	if err != nil {
		t.Fatalf("resolve Surface source: %v", err)
	}
	evidence, err := collectSurfaceProcessEvidence(smokeOptions{Mode: "headless", ReportPath: reportPath, SourcePath: source})
	if err != nil {
		t.Fatalf("collectSurfaceProcessEvidence(headless): %v", err)
	}
	traceArtifact := artifactByKind(evidence.Artifacts, "runner-trace")
	if traceArtifact == nil {
		t.Fatalf("artifacts = %#v, want headless runner trace artifact", evidence.Artifacts)
	}
	if evidence.ArtifactScan.FilesChecked < 2 {
		t.Fatalf("artifact scan = %#v, want component app and runner trace checked", evidence.ArtifactScan)
	}
	traceFrames, err := readHeadlessSurfaceTrace(traceArtifact.Path)
	if err != nil {
		t.Fatalf("readHeadlessSurfaceTrace: %v", err)
	}
	if len(traceFrames) != 2 {
		t.Fatalf("trace frames = %#v, want deterministic pre/post frames", traceFrames)
	}
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	if traceFrames[0].Checksum != checksumRGBA(beforeFrame.Pixels) || traceFrames[1].Checksum != checksumRGBA(afterFrame.Pixels) {
		t.Fatalf("trace frames = %#v, want actual headless runner checksums", traceFrames)
	}
}

func TestBlockSystemRuntimeSmokeReportArtifactScanIncludesFrameArtifacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "surface-headless-block-system.json")
	cmd := exec.Command(testBinary,
		"-test.run=^TestSurfaceRuntimeSmokeMainHelperProcess$",
		"--",
		"--mode", "headless-block-system",
		"--source", "examples/surface_block_system.tetra",
		"--report", reportPath,
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SURFACE_RUNTIME_SMOKE_MAIN_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface-runtime-smoke main failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, raw)
	}
	actualFiles, err := countRegularFilesUnder(report.ArtifactScan.Root)
	if err != nil {
		t.Fatalf("count files under artifact_scan.root: %v", err)
	}
	if report.ArtifactScan.FilesChecked != actualFiles {
		t.Fatalf("artifact_scan.files_checked = %d, actual files under %s = %d", report.ArtifactScan.FilesChecked, report.ArtifactScan.Root, actualFiles)
	}
	if report.ArtifactScan.FilesChecked < 5 {
		t.Fatalf("artifact_scan.files_checked = %d, want component app, runner trace, and frame artifacts", report.ArtifactScan.FilesChecked)
	}
}

func TestSurfaceRuntimeSmokeMainHelperProcess(t *testing.T) {
	if os.Getenv("SURFACE_RUNTIME_SMOKE_MAIN_HELPER") != "1" {
		return
	}
	args := []string{"surface-runtime-smoke"}
	for i, arg := range os.Args {
		if arg == "--" {
			args = append(args, os.Args[i+1:]...)
			break
		}
	}
	if len(args) == 1 {
		t.Fatal("missing surface-runtime-smoke helper arguments")
	}
	os.Args = args
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
}

func countRegularFilesUnder(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func TestHeadlessCounterScenarioFrameChecksumIsDeterministic(t *testing.T) {
	first := runHeadlessCounterScenario()
	second := runHeadlessCounterScenario()
	if len(first.Frames) != len(second.Frames) {
		t.Fatalf("frame count changed: %d != %d", len(first.Frames), len(second.Frames))
	}
	for i := range first.Frames {
		if first.Frames[i].Checksum != second.Frames[i].Checksum {
			t.Fatalf("checksum %d changed: %s != %s", i, first.Frames[i].Checksum, second.Frames[i].Checksum)
		}
	}
}

func TestHeadlessCounterScenarioFrameChecksumComesFromRGBAFramebuffer(t *testing.T) {
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	scenario := runHeadlessCounterScenario()
	if len(scenario.Frames) != 2 {
		t.Fatalf("frames = %#v, want pre-event and post-event frames", scenario.Frames)
	}
	for i, frame := range []rgbaFrame{beforeFrame, afterFrame} {
		reported := scenario.Frames[i]
		if reported.Width != frame.Width || reported.Height != frame.Height || reported.Stride != frame.Stride {
			t.Fatalf("frame %d dimensions = %dx%d stride %d, want %dx%d stride %d", i+1, reported.Width, reported.Height, reported.Stride, frame.Width, frame.Height, frame.Stride)
		}
		if len(frame.Pixels) != frame.Stride*frame.Height {
			t.Fatalf("pixel buffer len = %d, want %d", len(frame.Pixels), frame.Stride*frame.Height)
		}
		want := checksumRGBA(frame.Pixels)
		if reported.Checksum != want {
			t.Fatalf("reported checksum %d = %s, want RGBA framebuffer checksum %s", i+1, reported.Checksum, want)
		}
	}
	if scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("pre/post frame checksums match: %#v", scenario.Frames)
	}
}

func TestBlockPaintScenarioProducesVisualPaintEvidence(t *testing.T) {
	if err := validateSmokeMode("headless-block-paint"); err != nil {
		t.Fatalf("validateSmokeMode(headless-block-paint) failed: %v", err)
	}
	scenario := runBlockPaintScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-paint",
		SourcePath: "examples/surface_block_paint_layers.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_paint_layers.tetra -o /tmp/surface-artifacts/surface-block-paint", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-paint", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-paint", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-paint"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.PaintLayers) < 5 || len(report.PaintCommands) < 5 {
		t.Fatalf("paint evidence = layers %#v commands %#v, want layered paint commands", report.PaintLayers, report.PaintCommands)
	}
	if !caseNamesContain(scenario.Cases, "block paint deterministic command order") || !caseNamesContain(scenario.Cases, "block paint unsupported blur rejected") {
		t.Fatalf("scenario cases = %#v, want paint command order and unsupported blur evidence", scenario.Cases)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("paint scenario frames = %#v, want visual checksum change across hover/pressed paint state", scenario.Frames)
	}
}

func TestBlockTextScenarioProducesTextMeasurementEvidence(t *testing.T) {
	scenario := runBlockTextScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-text",
		SourcePath: "examples/surface_block_text.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_text.tetra -o /tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-text", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-text"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.TextMeasurements) < 2 || len(report.FontFallbacks) == 0 || len(report.GlyphCaches) == 0 || len(report.TextRenderCommands) < 2 {
		t.Fatalf("text evidence = measurements %#v fallback %#v cache %#v commands %#v, want text engine evidence", report.TextMeasurements, report.FontFallbacks, report.GlyphCaches, report.TextRenderCommands)
	}
	if !caseNamesContain(scenario.Cases, "block text wrap ellipsis layout") || !caseNamesContain(scenario.Cases, "block text editable lifetime") {
		t.Fatalf("scenario cases = %#v, want wrap/ellipsis and editable lifetime evidence", scenario.Cases)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("text scenario frames = %#v, want text render checksum change", scenario.Frames)
	}
}

func TestBlockLayoutScenarioProducesLayoutConstraintEvidence(t *testing.T) {
	scenario := runBlockLayoutScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-layout",
		SourcePath: "examples/surface_block_layout.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_layout.tetra -o /tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-layout", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-layout"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.LayoutConstraints) < 4 || len(report.LayoutPasses) < 8 || len(report.LayoutScrolls) == 0 {
		t.Fatalf("layout evidence = constraints %#v passes %#v scrolls %#v, want constraint resolver evidence", report.LayoutConstraints, report.LayoutPasses, report.LayoutScrolls)
	}
	if !caseNamesContain(scenario.Cases, "block layout grid dock overlay scroll") || !caseNamesContain(scenario.Cases, "block layout resize constraints") {
		t.Fatalf("scenario cases = %#v, want grid/dock/overlay/scroll and resize evidence", scenario.Cases)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("layout scenario frames = %#v, want responsive layout checksum change", scenario.Frames)
	}
}

func TestBlockEventScenarioProducesEventFocusEvidence(t *testing.T) {
	scenario := runBlockEventScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-events",
		SourcePath: "examples/surface_block_events.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_events.tetra -o /tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-events", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-events"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.BlockEventRoutes) < 5 || len(report.BlockFocusTransitions) < 2 {
		t.Fatalf("event evidence = routes %#v focus %#v, want routed Block event/focus evidence", report.BlockEventRoutes, report.BlockFocusTransitions)
	}
	if !caseNamesContain(scenario.Cases, "block event disabled click rejected") || !caseNamesContain(scenario.Cases, "block focus tab order graph-derived") {
		t.Fatalf("scenario cases = %#v, want disabled click and graph-derived focus evidence", scenario.Cases)
	}
}

func TestBlockStateScenarioProducesSelectorResolverEvidence(t *testing.T) {
	scenario := runBlockStateScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-states",
		SourcePath: "examples/surface_block_states.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_states.tetra -o /tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-states", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-states"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.BlockStateSelectors) < 7 || len(report.BlockStateResolutions) < 8 {
		t.Fatalf("state evidence = selectors %#v resolutions %#v, want generic selector resolver evidence", report.BlockStateSelectors, report.BlockStateResolutions)
	}
	if !caseNamesContain(scenario.Cases, "block state hover fill override") || !caseNamesContain(scenario.Cases, "block state disabled error loading overrides") {
		t.Fatalf("scenario cases = %#v, want hover and disabled/error/loading state evidence", scenario.Cases)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("state scenario frames = %#v, want state-driven checksum change", scenario.Frames)
	}
}

func TestBlockMotionScenarioProducesTransitionEvidence(t *testing.T) {
	scenario := runBlockMotionScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-motion",
		SourcePath: "examples/surface_block_motion.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_motion.tetra -o /tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-motion", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-motion"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.MotionFrames) < 4 {
		t.Fatalf("motion frames = %#v, want deterministic transition and reduced-motion evidence", report.MotionFrames)
	}
	last := report.MotionFrames[len(report.MotionFrames)-1]
	if !last.ReducedMotion || last.Scheduled || !last.Settled {
		t.Fatalf("last motion frame = %#v, want reduced motion settled without scheduling", last)
	}
	if !caseNamesContain(scenario.Cases, "block motion opacity color transform frames") || !caseNamesContain(scenario.Cases, "block motion completion stops scheduling") {
		t.Fatalf("scenario cases = %#v, want transition and completion evidence", scenario.Cases)
	}
}

func TestBlockAssetScenarioProducesLocalAssetEvidence(t *testing.T) {
	scenario := runBlockAssetScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-assets",
		SourcePath: "examples/surface_block_assets.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_assets.tetra -o /tmp/surface-artifacts/surface-block-assets", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-assets", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-assets", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-assets"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockAssetManifest == nil || len(report.BlockAssetManifest.Assets) < 3 {
		t.Fatalf("asset manifest = %#v, want font/icon/image asset hashes", report.BlockAssetManifest)
	}
	if report.BlockAssetNetworkFetchAllowed {
		t.Fatalf("network fetch allowed, want local/embedded-only asset evidence")
	}
	if !report.BlockAssetCache.Bounded || report.BlockAssetCache.UsedBytes <= 0 {
		t.Fatalf("asset cache = %#v, want bounded cache use evidence", report.BlockAssetCache)
	}
	if len(report.BlockAssetDiagnostics) == 0 {
		t.Fatalf("asset diagnostics = %#v, want missing asset fallback and network rejection evidence", report.BlockAssetDiagnostics)
	}
	for _, want := range []string{"block asset icon tint evidence", "block asset image scale evidence", "block asset network url rejected"} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("asset scenario frames = %#v, want asset-driven checksum change", scenario.Frames)
	}
}

func TestBlockAccessibilityScenarioProducesGraphDerivedMetadataEvidence(t *testing.T) {
	scenario := runBlockAccessibilityScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-accessibility",
		SourcePath: "examples/surface_block_accessibility.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_accessibility.tetra -o /tmp/surface-artifacts/surface-block-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-accessibility", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-accessibility"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockAccessibilityTree == nil {
		t.Fatalf("block accessibility tree is nil, want Block-derived metadata evidence")
	}
	if report.BlockAccessibilityTree.PlatformHostIntegration || report.BlockAccessibilityTree.ScreenReaderEvidence != false {
		t.Fatalf("block accessibility claims = platform %t screen-reader %#v, want metadata-only scoped claims", report.BlockAccessibilityTree.PlatformHostIntegration, report.BlockAccessibilityTree.ScreenReaderEvidence)
	}
	if !intSlicesEqual(report.BlockAccessibilityTree.ReadingOrder, report.BlockGraph.AccessibilityOrder) {
		t.Fatalf("reading order = %#v, want block graph accessibility order %#v", report.BlockAccessibilityTree.ReadingOrder, report.BlockGraph.AccessibilityOrder)
	}
	for _, want := range []string{"block accessibility tree derived from block graph", "block accessibility screen-reader claim without platform proof rejected"} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
}

func TestBlockSystemScenarioProducesHeadlessGoldenChecksumEvidence(t *testing.T) {
	if err := validateSmokeMode("headless-block-system"); err != nil {
		t.Fatalf("validateSmokeMode(headless-block-system) failed: %v", err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: "headless-block-system", SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_block_system.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(headless-block-system) = %q, want examples/surface_block_system.tetra", got)
	}

	scenario := runBlockSystemScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-system"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockSystem == nil {
		t.Fatalf("block_system is nil, want headless Block golden/checksum evidence")
	}
	if report.BlockSystem.Schema != "tetra.surface.block-system.v1" || report.BlockSystem.QualityLevel != "deterministic-headless-block-system-v1" {
		t.Fatalf("block_system = %#v, want deterministic headless Block system schema", report.BlockSystem)
	}
	if len(report.BlockSystem.Frames) != len(report.Frames) {
		t.Fatalf("block_system frames = %d, report frames = %d", len(report.BlockSystem.Frames), len(report.Frames))
	}
	for i, frame := range report.BlockSystem.Frames {
		if frame.Checksum != report.Frames[i].Checksum || frame.GoldenChecksum != report.Frames[i].Checksum {
			t.Fatalf("block_system frame %d checksums = %#v, report frame = %#v", i, frame, report.Frames[i])
		}
		if frame.RepeatChecksum != frame.Checksum {
			t.Fatalf("block_system frame %d repeat checksum = %q, want %q", i, frame.RepeatChecksum, frame.Checksum)
		}
		if !frame.PaintEvidence || !frame.LayoutEvidence || !frame.AccessibilityEvidence {
			t.Fatalf("block_system frame %d = %#v, want paint/layout/accessibility evidence flags", i, frame)
		}
	}
	for _, want := range []string{
		"block system headless golden checksums",
		"block system deterministic repeat checksum",
		"block system missing frame checksum rejected",
		"block system nondeterministic checksum rejected",
		"block system missing paint evidence rejected",
		"block system missing layout evidence rejected",
		"block system missing accessibility evidence rejected",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
}

func TestBlockSystemScenarioWritesFrameArtifacts(t *testing.T) {
	dir := t.TempDir()
	scenario := runBlockSystemScenario()
	opt := smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface_block_system.tetra",
		ReportPath: filepath.Join(dir, "surface-headless-block-system.json"),
	}
	if err := attachBlockSystemFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachBlockSystemFrameArtifacts failed: %v", err)
	}
	for _, frame := range scenario.Frames {
		if frame.ArtifactPath == "" {
			t.Fatalf("frame %d missing artifact_path", frame.Order)
		}
		raw, err := os.ReadFile(frame.ArtifactPath)
		if err != nil {
			t.Fatalf("read frame artifact %s: %v", frame.ArtifactPath, err)
		}
		if got := checksumRGBA(raw); got != frame.Checksum {
			t.Fatalf("frame artifact %s checksum = %s, want %s", frame.ArtifactPath, got, frame.Checksum)
		}
	}
	for _, frame := range scenario.BlockSystem.Frames {
		if frame.ArtifactPath == "" {
			t.Fatalf("block_system frame %d missing artifact_path", frame.Order)
		}
	}
}

func TestWASM32WebBrowserCanvasBlockSystemUsesExistingFrameArtifacts(t *testing.T) {
	dir := t.TempDir()
	initialFrame := renderBlockSystemFrameSizedRGBA(320, 200, false)
	focusedFrame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	initialPath := filepath.Join(dir, "surface-browser-canvas-frame-order-1.rgba")
	focusedPath := filepath.Join(dir, "surface-browser-canvas-frame-order-5.rgba")
	if err := os.WriteFile(initialPath, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial frame artifact: %v", err)
	}
	if err := os.WriteFile(focusedPath, focusedFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write focused frame artifact: %v", err)
	}

	scenario := runWASM32WebBrowserCanvasBlockSystemScenario()
	scenario.Frames = []surface.FrameReport{
		{
			Order:        1,
			Width:        initialFrame.Width,
			Height:       initialFrame.Height,
			Stride:       initialFrame.Stride,
			Checksum:     checksumRGBA(initialFrame.Pixels),
			ArtifactPath: initialPath,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        focusedFrame.Width,
			Height:       focusedFrame.Height,
			Stride:       focusedFrame.Stride,
			Checksum:     checksumRGBA(focusedFrame.Pixels),
			ArtifactPath: focusedPath,
			Presented:    true,
		},
	}
	scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario("examples/surface_block_system.tetra", scenario.Frames)
	opt := smokeOptions{
		Mode:       "wasm32-web-browser-canvas-block-system",
		SourcePath: "examples/surface_block_system.tetra",
		ReportPath: filepath.Join(dir, "surface-wasm-block-system.json"),
	}
	if err := attachBlockSystemFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachBlockSystemFrameArtifacts failed: %v", err)
	}

	for _, frame := range scenario.BlockSystem.Frames {
		switch frame.Order {
		case 1:
			if frame.ArtifactPath != initialPath {
				t.Fatalf("block_system frame order 1 artifact_path = %q, want %q", frame.ArtifactPath, initialPath)
			}
		case 5:
			if frame.ArtifactPath != focusedPath {
				t.Fatalf("block_system frame order 5 artifact_path = %q, want %q", frame.ArtifactPath, focusedPath)
			}
		default:
			t.Fatalf("unexpected block_system frame order %d", frame.Order)
		}
	}
	syntheticPath := filepath.Join(surfaceRuntimeArtifactDir(opt), "surface-block-system-frame-order-1-initial.rgba")
	if _, err := os.Stat(syntheticPath); err == nil {
		t.Fatalf("synthetic wasm Block-system artifact exists at %s", syntheticPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat synthetic wasm Block-system artifact %s: %v", syntheticPath, err)
	}
}

func TestMorphScenarioProducesHeadlessCapsuleEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_command_palette.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != source {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want %s", mode, got, source)
	}
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Morph == nil {
		t.Fatalf("morph is nil, want Morph capsule evidence")
	}
	if report.Morph.Schema != "tetra.surface.morph.v1" || report.Morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		t.Fatalf("morph = %#v, want Morph v1 deterministic headless capsule evidence", report.Morph)
	}
	if report.Morph.TokenGraph == nil || report.Morph.TokenGraph.SourceOfTruth != "capsule" || !report.Morph.TokenGraph.ExplicitImports || !report.Morph.TokenGraph.NoGlobalCascade {
		t.Fatalf("morph token graph = %#v, want P07 capsule source-of-truth boundary evidence", report.Morph.TokenGraph)
	}
	if len(report.Morph.TokenGraph.Tokens) < 22 || len(report.Morph.TokenGraph.DensityDPI) != 3 || !report.Morph.TokenGraph.Diagnostics.CSSRuntimeRejected {
		t.Fatalf("morph token graph = %#v, want P07 typed tokens, density mapping, and diagnostics", report.Morph.TokenGraph)
	}
	if len(report.Morph.Recipes) != 19 || len(report.Morph.RecipeExpansions) != 19 || len(report.Morph.RecipeApps) != 7 {
		t.Fatalf("morph recipes=%d expansions=%d apps=%d, want P08 recipe authoring suite",
			len(report.Morph.Recipes), len(report.Morph.RecipeExpansions), len(report.Morph.RecipeApps))
	}
	if report.BlockSystem == nil || report.BlockSystem.Source != source || report.BlockGraph.Source != source {
		t.Fatalf("Block evidence sources = block_system %#v block_graph %#v, want Morph source %s", report.BlockSystem, report.BlockGraph, source)
	}
	if !caseNamesContain(report.Cases, "morph recipes expand to Block graph") {
		t.Fatalf("cases = %#v, want Morph recipe expansion evidence", report.Cases)
	}
}

func TestMorphScenarioProducesBlockSceneSnapshotEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	snapshot := report.BlockSceneSnapshot
	if snapshot == nil {
		t.Fatalf("block_scene_snapshot is nil, want rich renderable Block scene snapshot evidence")
	}
	if snapshot.Source != source || snapshot.QualityLevel != "rich-renderable-block-scene-v1" {
		t.Fatalf("block_scene_snapshot = %#v, want Morph source and rich renderable quality", snapshot)
	}
	if snapshot.CompactPropsOnly || len(snapshot.Nodes) == 0 || snapshot.NodeCount != len(snapshot.Nodes) {
		t.Fatalf("block_scene_snapshot compact=%t node_count=%d len(nodes)=%d, want rich node evidence", snapshot.CompactPropsOnly, snapshot.NodeCount, len(snapshot.Nodes))
	}
	coverage := snapshot.SpecCoverage
	if !coverage.Layout || !coverage.Paint || !coverage.Text || !coverage.Image || !coverage.Input || !coverage.Event || !coverage.State || !coverage.Motion || !coverage.Accessibility {
		t.Fatalf("block_scene_snapshot spec coverage = %#v, want all rich specs preserved", coverage)
	}
	for _, want := range []string{
		"block scene snapshot preserves rich visual specs",
		"block scene compact BlockProps-only evidence rejected",
		"block scene non-Block core primitive rejected",
		"block scene missing rich spec coverage rejected",
	} {
		if !caseNamesContain(report.Cases, want) {
			t.Fatalf("cases = %#v, want %q", report.Cases, want)
		}
	}
}

func TestMorphScenarioProducesRenderCommandStreamEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	stream := report.RenderCommandStream
	if stream == nil {
		t.Fatalf("render_command_stream is nil, want source-linked stream from Morph-authored Block scene")
	}
	if report.BlockSceneSnapshot == nil {
		t.Fatalf("block_scene_snapshot is nil")
	}
	if stream.Source != source || stream.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash || !stream.DerivedFromBlockSceneSnapshot {
		t.Fatalf("render_command_stream = %#v, want source %s and block_scene_hash %s", stream, source, report.BlockSceneSnapshot.BlockSceneHash)
	}
	if !stream.SourceLinked || stream.HandcraftedFixture || stream.CommandCount != len(stream.Commands) || stream.CommandCount < 10 {
		t.Fatalf("render_command_stream = %#v, want source-linked non-fixture command evidence", stream)
	}
	if len(report.Frames) == 0 || stream.FrameChecksum != report.Frames[0].Checksum {
		t.Fatalf("render_command_stream frame_checksum = %q, want first frame checksum", stream.FrameChecksum)
	}
	for _, command := range []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"} {
		if !renderCommandStreamHasCommand(stream.Commands, command) {
			t.Fatalf("render_command_stream commands = %#v, want %s", stream.Commands, command)
		}
	}
	for _, command := range stream.Commands {
		if command.Source != source || command.Recipe == "" || command.BlockID <= 0 {
			t.Fatalf("render command = %#v, want source, recipe, and block_id links", command)
		}
	}
}

func TestMorphScenarioBuildsRenderedBeautyReportFromRuntimeAndVisualEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)

	mrb, err := buildMorphRenderedBeautyReport("reports/surface/morph-runtime.json", report, visual, "headless-morph")
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if report.Frames[0].ArtifactPath == "" || report.Frames[0].Producer != "app" || report.Frames[0].EvidenceRole != "product_visual" {
		t.Fatalf("morph frame provenance = %#v, want app-produced product_visual artifact", report.Frames[0])
	}
	if mrb.Schema != surface.MorphRenderedBeautyReportSchemaV1 || mrb.Target != "headless" || mrb.ScenarioName != "headless-morph" {
		t.Fatalf("MRB report identity = %#v, want schema, target, and scenario", mrb)
	}
	if mrb.MorphEvidence.Source != source || mrb.MorphEvidence.TokenCount == 0 || len(mrb.MorphEvidence.RecipeNames) == 0 {
		t.Fatalf("morph evidence = %#v, want source, token coverage, and recipe coverage", mrb.MorphEvidence)
	}
	if mrb.BlockSceneSnapshot.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		t.Fatalf("block scene hash = %q, want %q", mrb.BlockSceneSnapshot.BlockSceneHash, report.BlockSceneSnapshot.BlockSceneHash)
	}
	if mrb.RenderCommandStream.CommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		t.Fatalf("render command stream hash = %q, want %q", mrb.RenderCommandStream.CommandStreamHash, report.RenderCommandStream.CommandStreamHash)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" || !mrb.RendererStableProof.RendererOwned || mrb.RendererStableProof.BridgeOwnedPixels || !mrb.RendererStableProof.DerivedFromRenderCommandStream || !mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf("renderer stable proof = %#v, want renderer-owned command-stream-derived proof", mrb.RendererStableProof)
	}
	if mrb.PixelEvidence.FrameProducer != "app" || mrb.PixelEvidence.AppSource != source || mrb.PixelEvidence.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		t.Fatalf("pixel evidence = %#v, want app-produced source-linked pixel chain", mrb.PixelEvidence)
	}
}

func TestApplyMorphRenderedBeautyProductSignoffRequiresCleanRendererOwnedProof(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)

	mrb, err := buildMorphRenderedBeautyReport("reports/surface/morph-runtime.json", report, visual, "headless-morph")
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	mrb.GitDirty = false
	if err := applyMorphRenderedBeautyProductSignoff(&mrb, true, true); err != nil {
		t.Fatalf("applyMorphRenderedBeautyProductSignoff rejected clean renderer-owned proof: %v", err)
	}
	if !mrb.ProductClaim || !mrb.FinalSignoff {
		t.Fatalf("MRB signoff = product_claim:%t final_signoff:%t, want true/true", mrb.ProductClaim, mrb.FinalSignoff)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue rejected signed MRB report: %v", err)
	}

	mrb.GitDirty = true
	mrb.ProductClaim = false
	mrb.FinalSignoff = false
	err = applyMorphRenderedBeautyProductSignoff(&mrb, true, true)
	if err == nil {
		t.Fatalf("expected dirty product signoff to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dirty") {
		t.Fatalf("error = %v, want dirty diagnostic", err)
	}
}

func TestMorphFlagshipScenarioProducesRenderedBeautyReport(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface_morph_rendered_studio_shell.tetra"
	rawSource, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", source)))
	if err != nil {
		t.Fatalf("read flagship Morph source: %v", err)
	}
	if strings.Contains(string(rawSource), "import lib.core.draw") || strings.Contains(string(rawSource), "func draw(") {
		t.Fatalf("flagship Morph source must not use manual draw authoring")
	}

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph-flagship.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_rendered_studio_shell.tetra -o /tmp/surface-artifacts/surface-morph-rendered-studio-shell", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph --source examples/surface_morph_rendered_studio_shell.tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-rendered-studio-shell"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Source != source || report.Morph == nil || report.Morph.Source != source || report.BlockSceneSnapshot.Source != source || report.RenderCommandStream.Source != source {
		t.Fatalf("flagship evidence sources = report %q morph %#v scene %#v stream %#v, want %s", report.Source, report.Morph, report.BlockSceneSnapshot, report.RenderCommandStream, source)
	}
	if report.BlockSceneSnapshot.NodeCount < 18 || report.Morph == nil || len(report.Morph.RecipeExpansions) < 16 {
		t.Fatalf("flagship scene coverage nodes=%d expansions=%d, want contract-scale Morph flagship evidence", report.BlockSceneSnapshot.NodeCount, len(report.Morph.RecipeExpansions))
	}
	for _, want := range []string{
		"app.shell@1",
		"nav.item@1",
		"toolbar@1",
		"split.pane@1",
		"status.bar@1",
		"command.item@1",
		"settings.form@1",
		"log.row@1",
		"metric.tile@1",
		"toast.notification@1",
		"dialog.panel@1",
		"empty.state@1",
		"error.panel@1",
	} {
		if !morphRecipeNamesContain(report.Morph.Recipes, want) {
			t.Fatalf("flagship Morph recipes missing %q in %#v", want, report.Morph.Recipes)
		}
	}
	for _, want := range []string{"DashboardShell", "ProfilesActions", "CommandPalette", "SettingsForm", "LogsOutput", "DiagnosticsError", "StatusBar", "BlockedDialog"} {
		if !blockSceneSnapshotHasNode(report.BlockSceneSnapshot, want) {
			t.Fatalf("flagship Block scene missing %q in %#v", want, report.BlockSceneSnapshot.Nodes)
		}
	}
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/flagship-morph-runtime.json", report)
	mrb, err := buildMorphRenderedBeautyReport("reports/surface/flagship-morph-runtime.json", report, visual, morphRenderedBeautyScenarioName(opt))
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.MorphEvidence.Source != source || mrb.ScenarioName != mode+":"+source || mrb.PixelEvidence.AppSource != source {
		t.Fatalf("flagship MRB source/scenario = %#v pixel %#v, want %s", mrb.MorphEvidence, mrb.PixelEvidence, source)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" || !mrb.RendererStableProof.RendererOwned || mrb.RendererStableProof.BridgeOwnedPixels || !mrb.RendererStableProof.DerivedFromRenderCommandStream || !mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf("flagship renderer stable proof = %#v, want renderer-owned command-stream-derived proof", mrb.RendererStableProof)
	}
}

func TestWASM32WebBrowserCanvasMorphScenarioUsesBrowserCanvasPixelEvidence(t *testing.T) {
	const mode = "wasm32-web-browser-canvas-morph"
	const source = "examples/surface_morph_rendered_studio_shell.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}

	dir := t.TempDir()
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	focusedFrame := renderMorphStudioShellFrameRGBA(320, 200, true)
	initialArtifact := filepath.Join(dir, "surface-browser-canvas-frame-order-1.rgba")
	focusedArtifact := filepath.Join(dir, "surface-browser-canvas-frame-order-5.rgba")
	if err := os.WriteFile(initialArtifact, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial browser canvas artifact: %v", err)
	}
	if err := os.WriteFile(focusedArtifact, focusedFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write focused browser canvas artifact: %v", err)
	}
	initialChecksum := checksumRGBA(initialFrame.Pixels)
	focusedChecksum := checksumRGBA(focusedFrame.Pixels)

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(dir, "surface-wasm-browser-canvas-morph.json"),
	}
	browserFrames := []surface.FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: initialChecksum, ArtifactPath: initialArtifact, Presented: true},
		{Order: 5, Width: 320, Height: 200, Stride: 1280, Checksum: focusedChecksum, ArtifactPath: focusedArtifact, Presented: true},
	}
	if err := applyMorphTargetRuntimeFrameEvidence(opt, &scenario, browserFrames); err != nil {
		t.Fatalf("applyMorphTargetRuntimeFrameEvidence: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_morph_rendered_studio_shell.tetra -o /tmp/surface-artifacts/surface-morph-rendered-studio-shell.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=studio-shell wasm=/tmp/surface-artifacts/surface-morph-rendered-studio-shell.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-morph-rendered-studio-shell.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium studio-shell fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom <surface-browser-canvas-file-runner scenario=studio-shell>", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
	}, cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "wasm32-web" || report.Runtime != "surface-wasm32-web" || !report.HostEvidence.BrowserCanvas || !report.HostEvidence.BrowserInput {
		t.Fatalf("target host evidence = target %q runtime %q host %#v, want browser-canvas wasm Morph evidence", report.Target, report.Runtime, report.HostEvidence)
	}
	if report.RenderCommandStream.Renderer != "browser-canvas-rgba" || report.RenderCommandStream.FrameChecksum != initialChecksum {
		t.Fatalf("render_command_stream = %#v, want browser renderer and first browser canvas checksum %s", report.RenderCommandStream, initialChecksum)
	}
	if report.Frames[0].ArtifactPath != initialArtifact || report.Frames[0].Producer != "app" || report.Frames[0].EvidenceRole != "product_visual" {
		t.Fatalf("browser Morph frame provenance = %#v, want app-produced product visual browser canvas artifact", report.Frames[0])
	}

	const runtimeReportPath = "reports/surface/wasm-browser-canvas-morph-runtime.json"
	visual := morphRenderedBeautyVisualReportForTest(runtimeReportPath, report)
	visual.RequiredTargets = []string{"wasm32-web-browser-canvas"}
	visual.Apps[0].Targets[0].Target = "wasm32-web-browser-canvas"
	visual.Apps[0].Targets[0].Renderer = "browser-canvas-rgba"
	mrb, err := buildMorphRenderedBeautyReport(runtimeReportPath, report, visual, morphRenderedBeautyScenarioName(opt))
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.Target != "wasm32-web-browser-canvas" || mrb.PixelEvidence.FrameArtifact != initialArtifact || mrb.PixelEvidence.FrameChecksum != normalizePrefixedSHA256(initialChecksum) {
		t.Fatalf("MRB target/pixel evidence = target %q pixel %#v, want browser canvas Morph pixel evidence", mrb.Target, mrb.PixelEvidence)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" || !mrb.RendererStableProof.RendererOwned || mrb.RendererStableProof.BridgeOwnedPixels || !mrb.RendererStableProof.DerivedFromRenderCommandStream || !mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf("browser canvas renderer stable proof = %#v, want renderer-owned command-stream-derived target proof", mrb.RendererStableProof)
	}
}

func TestLinuxX64RealWindowMorphScenarioUsesAppProducedPixelEvidence(t *testing.T) {
	const mode = "linux-x64-real-window-morph"
	const source = "examples/surface_morph_rendered_studio_shell.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}

	dir := t.TempDir()
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	activeFrame := renderMorphStudioShellFrameRGBA(400, 240, true)
	initialArtifact := filepath.Join(dir, "surface-morph-real-window-frame-order-1.rgba")
	if err := os.WriteFile(initialArtifact, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial linux Morph frame artifact: %v", err)
	}
	activeArtifact := filepath.Join(dir, "surface-morph-real-window-frame-order-5.rgba")
	if err := os.WriteFile(activeArtifact, activeFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write linux Morph frame artifact: %v", err)
	}
	initialChecksum := checksumRGBA(initialFrame.Pixels)
	activeChecksum := checksumRGBA(activeFrame.Pixels)

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(dir, "surface-linux-real-window-morph.json"),
	}
	frames := []surface.FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: initialChecksum, ArtifactPath: initialArtifact, Presented: true},
		{Order: 5, Width: 400, Height: 240, Stride: 1600, Checksum: activeChecksum, ArtifactPath: activeArtifact, Presented: true},
	}
	if err := applyMorphTargetRuntimeFrameEvidence(opt, &scenario, frames); err != nil {
		t.Fatalf("applyMorphTargetRuntimeFrameEvidence: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_rendered_studio_shell.tetra -o /tmp/surface-artifacts/surface-morph-rendered-studio-shell", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface linux-x64 Morph app-presented frame probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell-present-probe", Ran: true, Pass: true, ExitCode: intPtr(-1), ExpectedExitCode: intPtr(-1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-rendered-studio-shell-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 real-window runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, componentAppArtifacts("/tmp/surface-artifacts/surface-morph-rendered-studio-shell"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" {
		t.Fatalf("target/runtime = %s/%s, want linux-x64/surface-linux-x64", report.Target, report.Runtime)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" || report.HostEvidence.Backend != "wayland-shm-rgba" || !report.HostEvidence.Framebuffer || !report.HostEvidence.RealWindow || !report.HostEvidence.NativeInput {
		t.Fatalf("host evidence = %#v, want linux-x64 real-window native-input evidence", report.HostEvidence)
	}
	if report.RenderCommandStream == nil || report.RenderCommandStream.Renderer != "wayland-shm-rgba" {
		t.Fatalf("render_command_stream = %#v, want wayland-shm-rgba", report.RenderCommandStream)
	}
	if len(report.Frames) != 2 {
		t.Fatalf("frames = %#v, want initial and active Linux real-window Morph product frames", report.Frames)
	}
	frame := report.Frames[1]
	if frame.Producer != "app" || frame.EvidenceRole != "product_visual" || frame.Precomputed || frame.ArtifactPath != activeArtifact || frame.Checksum != activeChecksum {
		t.Fatalf("frame provenance = %#v, want app product_visual non-precomputed artifact evidence", frame)
	}
	if report.BlockSystem != nil {
		t.Fatalf("block_system = %#v, want nil for target-owned Morph product frame evidence", report.BlockSystem)
	}
	if !caseNamesContain(report.Cases, "linux-x64 real-window Morph rendered beauty app frame readback") {
		t.Fatalf("cases = %#v, want Linux Morph app frame readback evidence", report.Cases)
	}

	const runtimeReportPath = "reports/surface/linux-real-window-morph-runtime.json"
	visual := morphRenderedBeautyVisualReportForTest(runtimeReportPath, report)
	visual.RequiredTargets = []string{"linux-x64-real-window"}
	visual.Apps[0].Targets[0].Target = "linux-x64-real-window"
	visual.Apps[0].Targets[0].Renderer = "wayland-shm-rgba"
	mrb, err := buildMorphRenderedBeautyReport(runtimeReportPath, report, visual, morphRenderedBeautyScenarioName(opt))
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.Target != "linux-x64-real-window" || mrb.PixelEvidence.FrameArtifact != initialArtifact || mrb.PixelEvidence.FrameChecksum != normalizePrefixedSHA256(initialChecksum) {
		t.Fatalf("MRB target/pixel evidence = target %q pixel %#v, want Linux real-window Morph pixel evidence", mrb.Target, mrb.PixelEvidence)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" || !mrb.RendererStableProof.RendererOwned || mrb.RendererStableProof.BridgeOwnedPixels || !mrb.RendererStableProof.DerivedFromRenderCommandStream || !mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf("Linux renderer stable proof = %#v, want renderer-owned command-stream-derived target proof", mrb.RendererStableProof)
	}
}

func TestBuildMorphRenderedBeautyReportRejectsMissingPixelGoldenFrame(t *testing.T) {
	const source = "examples/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-morph",
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_morph_command_palette.tetra -o /tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-morph-command-palette", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-morph", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-morph-command-palette"), cleanArtifactScan(2), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)
	visual.Apps[0].Targets[0].Frames = nil

	_, err := buildMorphRenderedBeautyReport("reports/surface/morph-runtime.json", report, visual, "headless-morph")
	if err == nil {
		t.Fatalf("expected missing pixel golden frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "pixel golden") {
		t.Fatalf("error = %v, want pixel golden diagnostic", err)
	}
}

func morphRenderedBeautyVisualReportForTest(runtimeReportPath string, report surface.Report) surface.VisualRegressionReport {
	frameChecksum := normalizePrefixedSHA256(report.RenderCommandStream.FrameChecksum)
	goldenChecksum := "sha256:" + strings.Repeat("9", 64)
	return surface.VisualRegressionReport{
		Schema:          surface.VisualRegressionSchemaV1,
		Status:          "pass",
		GitHead:         report.Morph.GitHead,
		GoldenSet:       "surface-morph-rendered-beauty-v1",
		GoldenHash:      "sha256:" + strings.Repeat("8", 64),
		RequiredTargets: []string{"headless"},
		RequiredSources: []string{report.Source},
		Apps: []surface.VisualRegressionAppReport{{
			Name:         "surface-morph-command-palette",
			Source:       report.Source,
			ReferenceApp: true,
			Targets: []surface.VisualRegressionTargetReport{{
				Target:                "headless",
				RuntimeReport:         runtimeReportPath,
				RuntimeSchema:         surface.SchemaV1,
				GitHead:               report.Morph.GitHead,
				GoldenGitHead:         report.Morph.GitHead,
				Renderer:              report.RenderCommandStream.Renderer,
				BlockGraphEvidence:    true,
				TokenThemeEvidence:    true,
				LayoutEvidence:        true,
				AccessibilityEvidence: true,
				PerformanceEvidence:   true,
				Frames: []surface.VisualRegressionFrameReport{{
					Order:                 1,
					Label:                 "initial",
					Width:                 report.Frames[0].Width,
					Height:                report.Frames[0].Height,
					Stride:                report.Frames[0].Stride,
					Checksum:              frameChecksum,
					GoldenChecksum:        goldenChecksum,
					ArtifactPath:          morphRenderedBeautyFrameArtifactPathForTest(report),
					ArtifactSHA256:        frameChecksum,
					ArtifactFormat:        "rgba",
					GoldenArtifactPath:    "reports/surface/morph-rendered-beauty/headless/golden.rgba",
					GoldenArtifactSHA256:  goldenChecksum,
					DiffPixels:            0,
					DiffRatioMilli:        0,
					MaxChannelDelta:       0,
					TolerancePixels:       4,
					ToleranceRatioMilli:   1,
					ToleranceChannelDelta: 1,
					Pass:                  true,
				}},
			}},
		}},
		NegativeGuards: surface.VisualRegressionNegativeGuardsReport{
			ScreenshotOnlyRejected:           true,
			StaleGoldenRejected:              true,
			MajorDriftRejected:               true,
			MissingBlockGraphRejected:        true,
			MissingLayoutRejected:            true,
			MissingAccessibilityRejected:     true,
			MissingPerformanceRejected:       true,
			SelfGoldenRejected:               true,
			MetadataChecksumRejected:         true,
			FixtureFrameOnlyRejected:         true,
			MissingPNGOrRGBAArtifactRejected: true,
		},
	}
}

func morphRenderedBeautyFrameArtifactPathForTest(report surface.Report) string {
	if len(report.Frames) > 0 && strings.TrimSpace(report.Frames[0].ArtifactPath) != "" {
		return report.Frames[0].ArtifactPath
	}
	return "reports/surface/morph-rendered-beauty/headless/current.rgba"
}

func morphRecipeNamesContain(recipes []surface.MorphRecipeReport, want string) bool {
	for _, recipe := range recipes {
		if recipe.Name == want {
			return true
		}
	}
	return false
}

func blockSceneSnapshotHasNode(snapshot *surface.BlockSceneSnapshotReport, want string) bool {
	if snapshot == nil {
		return false
	}
	for _, node := range snapshot.Nodes {
		if node.Name == want || node.Recipe == want {
			return true
		}
	}
	return false
}

func renderCommandStreamHasCommand(commands []surface.RenderCommandReport, want string) bool {
	for _, command := range commands {
		if command.Command == want {
			return true
		}
	}
	return false
}
