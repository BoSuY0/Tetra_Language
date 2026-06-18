package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
	"tetra_language/internal/outputformat"
	"tetra_language/tools/validators/surface"
	"tetra_language/tools/validators/surfaceinspector"
	"time"
)

// ---- doctor.go ----

type doctorReport struct {
	Status string        `json:"status"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "doctor accepts at most one path")
		return 2
	}
	report := doctorReport{}
	if fs.NArg() == 1 {
		report = buildProjectDoctorReport(fs.Arg(0))
	} else if ctx, err := discoverCLIProject("."); err == nil && ctx != nil && ctx.Found {
		report = buildProjectDoctorReport(ctx.Root)
	} else {
		report = buildDoctorReport()
	}
	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "Tetra doctor: %s\n", report.Status)
		for _, check := range report.Checks {
			if check.Detail == "" {
				fmt.Fprintf(stdout, "  %s: %s\n", check.Name, check.Status)
			} else {
				fmt.Fprintf(stdout, "  %s: %s (%s)\n", check.Name, check.Status, check.Detail)
			}
		}
	case "json", "toon":
		if err := outputformat.WriteStructured(stdout, *format, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
	if report.Status != "pass" {
		return 1
	}
	return 0
}

func buildProjectDoctorReport(start string) doctorReport {
	ctx, err := discoverCLIProject(start)
	if err != nil {
		return doctorReport{
			Status: "fail",
			Checks: []doctorCheck{failCheck("project capsule", err.Error())},
		}
	}
	if ctx == nil || !ctx.Found {
		return doctorReport{
			Status: "fail",
			Checks: []doctorCheck{failCheck("project capsule", "not found")},
		}
	}
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("project root", ctx.Root),
		passCheck("project capsule", ctx.CapsulePath),
		passCheck("project entry", ctx.EntryPath),
	}
	sourcePaths := existingProjectSourcePaths(ctx)
	if len(sourcePaths) == 0 {
		checks = append(checks, failCheck("project source roots", "no existing source roots"))
	} else {
		var rels []string
		for _, path := range sourcePaths {
			rel, err := filepath.Rel(ctx.Root, path)
			if err != nil {
				rels = append(rels, filepath.ToSlash(path))
				continue
			}
			rels = append(rels, filepath.ToSlash(rel))
		}
		checks = append(checks, passCheck("project source roots", strings.Join(rels, ", ")))
	}
	if len(ctx.DependencyRoots) == 0 {
		checks = append(checks, passCheck("project dependencies", "none"))
	} else {
		checks = append(
			checks,
			passCheck("project dependencies", fmt.Sprintf("%d root(s)", len(ctx.DependencyRoots))),
		)
	}
	if ctx.LockPath == "" {
		checks = append(
			checks,
			passCheck(
				"project lock",
				"not present; run "+projectSyncRepairCommand(ctx.Root, "", false),
			),
		)
	} else if err := validateDiscoveredProjectLock(ctx, ""); err != nil {
		checks = append(checks, failCheck("project lock", err.Error()))
	} else {
		checks = append(checks, passCheck("project lock", ctx.LockPath))
	}
	return doctorReport{Status: doctorStatus(checks), Checks: checks}
}

func buildDoctorReport() doctorReport {
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("supported targets", strings.Join(ctarget.SupportedTriples(), ", ")),
		passCheck("build-only targets", strings.Join(ctarget.BuildOnlyTriples(), ", ")),
		passCheck("planned targets", strings.Join(ctarget.PlannedTriples(), ", ")),
	}
	root, err := findRepoRoot()
	if err != nil {
		checks = append(checks, failCheck("repo root", err.Error()))
		return doctorReport{Status: doctorStatus(checks), Checks: checks}
	}
	return buildDoctorReportForRootWithChecks(root, checks)
}

func buildDoctorReportForRoot(root string) doctorReport {
	checks := []doctorCheck{
		passCheck("version", compiler.Version()),
		passCheck("supported targets", strings.Join(ctarget.SupportedTriples(), ", ")),
		passCheck("build-only targets", strings.Join(ctarget.BuildOnlyTriples(), ", ")),
		passCheck("planned targets", strings.Join(ctarget.PlannedTriples(), ", ")),
	}
	return buildDoctorReportForRootWithChecks(root, checks)
}

func buildDoctorReportForRootWithChecks(root string, checks []doctorCheck) doctorReport {
	checks = append(checks,
		passCheck("repo root", root),
		pathCheck(root, "__rt/actors_sysv.tetra"),
		pathCheck(root, "__rt/actors_win64.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_sysv.tetra"),
		pathCheck(root, "compiler/selfhostrt/actors_win64.tetra"),
		pathCheck(root, "examples/flow/flow_hello.tetra"),
		pathCheck(root, "docs/generated/manifest.json"),
		manifestVersionCheck(root),
		manifestSurfaceCheck(root),
		smokeSourcesCheck(root),
		runtimeExportsCheck(root),
		targetMetadataCheck(),
		toolingCommandsCheck(),
	)
	return doctorReport{Status: doctorStatus(checks), Checks: checks}
}

func manifestVersionCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	var manifest struct {
		CompilerVersion string `json:"compiler_version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest version", err.Error())
	}
	if manifest.CompilerVersion != compiler.Version() {
		return failCheck(
			"docs manifest version",
			fmt.Sprintf("got %s want %s", manifest.CompilerVersion, compiler.Version()),
		)
	}
	return passCheck("docs manifest version", manifest.CompilerVersion)
}

func manifestSurfaceCheck(root string) doctorCheck {
	path := filepath.Join(root, filepath.FromSlash("docs/generated/manifest.json"))
	raw, err := os.ReadFile(path)
	if err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	var manifest struct {
		Formats []struct {
			Extension string `json:"extension,omitempty"`
			FileName  string `json:"file_name,omitempty"`
			Role      string `json:"role"`
			Primary   bool   `json:"primary,omitempty"`
			Legacy    bool   `json:"legacy,omitempty"`
		} `json:"formats"`
		Targets []struct {
			Triple string `json:"triple"`
		} `json:"targets"`
		RuntimeABI struct {
			ActorsSupportedTargets []string `json:"actors_supported_targets"`
			ActorsRequiredSymbols  []string `json:"actors_required_symbols"`
			TimeRequiredSymbols    []string `json:"time_required_symbols"`
		} `json:"runtime_abi"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return failCheck("docs manifest surface", err.Error())
	}
	formatKeys := map[string]bool{}
	var sourcePrimary, sourceLegacy bool
	for _, format := range manifest.Formats {
		key := format.Extension
		if key == "" {
			key = format.FileName
		}
		formatKeys[key] = true
		if key == compiler.T4SourceExtension && format.Role == "source" && format.Primary {
			sourcePrimary = true
		}
		if key == compiler.LegacyTetraSourceExtension && format.Role == "source" && format.Legacy {
			sourceLegacy = true
		}
	}
	requiredFormats := []string{
		compiler.T4SourceExtension,
		compiler.TodexFragmentExtension,
		compiler.T4SeedExtension,
		compiler.T4InterfaceExtension,
		compiler.T4ProofExtension,
		compiler.T4ReplayExtension,
		compiler.T4QuestExtension,
		compiler.NeedMapExtension,
		compiler.SemanticLockFileName,
	}
	for _, key := range requiredFormats {
		if !formatKeys[key] {
			return failCheck("docs manifest surface", "missing format "+key)
		}
	}
	if !sourcePrimary || !sourceLegacy {
		return failCheck("docs manifest surface", "missing source format primary/legacy markers")
	}
	var targetTriples []string
	for _, target := range manifest.Targets {
		targetTriples = append(targetTriples, target.Triple)
	}
	var buildableTargets []string
	for _, target := range ctarget.AllBuildable() {
		buildableTargets = append(buildableTargets, target.Triple)
	}
	if !sameStringSet(targetTriples, buildableTargets) {
		return failCheck(
			"docs manifest surface",
			fmt.Sprintf(
				"targets got %s want %s",
				strings.Join(sortedDoctorStrings(targetTriples), ", "),
				strings.Join(sortedDoctorStrings(buildableTargets), ", "),
			),
		)
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsSupportedTargets, ctarget.ActorRuntimeTriples()) {
		return failCheck(
			"docs manifest surface",
			fmt.Sprintf(
				"actors targets got %s want %s",
				strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsSupportedTargets), ", "),
				strings.Join(sortedDoctorStrings(ctarget.ActorRuntimeTriples()), ", "),
			),
		)
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsRequiredSymbols, actorRuntimeSymbols()) {
		return failCheck(
			"docs manifest surface",
			fmt.Sprintf(
				"runtime symbols got %s want %s",
				strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsRequiredSymbols), ", "),
				strings.Join(sortedDoctorStrings(actorRuntimeSymbols()), ", "),
			),
		)
	}
	if !sameStringSet(manifest.RuntimeABI.TimeRequiredSymbols, timeRuntimeSymbols()) {
		return failCheck(
			"docs manifest surface",
			fmt.Sprintf(
				"time runtime symbols got %s want %s",
				strings.Join(sortedDoctorStrings(manifest.RuntimeABI.TimeRequiredSymbols), ", "),
				strings.Join(sortedDoctorStrings(timeRuntimeSymbols()), ", "),
			),
		)
	}
	return passCheck(
		"docs manifest surface",
		fmt.Sprintf(
			"%d formats, %d targets, %d runtime symbols",
			len(manifest.Formats),
			len(targetTriples),
			len(actorRuntimeSymbols())+len(timeRuntimeSymbols()),
		),
	)
}

func smokeSourcesCheck(root string) doctorCheck {
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	var missing []string
	var duplicates []string
	cases := smokeCases(true)
	for _, c := range cases {
		if seenNames[c.name] {
			duplicates = append(duplicates, "name:"+c.name)
		}
		seenNames[c.name] = true
		if seenSources[c.srcPath] {
			duplicates = append(duplicates, "src:"+c.srcPath)
		}
		seenSources[c.srcPath] = true
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.srcPath))); err != nil {
			missing = append(missing, c.srcPath)
		}
	}
	if len(missing) > 0 || len(duplicates) > 0 {
		sort.Strings(missing)
		sort.Strings(duplicates)
		parts := []string{}
		if len(missing) > 0 {
			parts = append(parts, "missing "+strings.Join(missing, ", "))
		}
		if len(duplicates) > 0 {
			parts = append(parts, "duplicate "+strings.Join(duplicates, ", "))
		}
		return failCheck("smoke sources", strings.Join(parts, "; "))
	}
	return passCheck("smoke sources", fmt.Sprintf("%d sources", len(cases)))
}

func runtimeExportsCheck(root string) doctorCheck {
	paths := []string{
		"__rt/actors_sysv.tetra",
		"__rt/actors_win64.tetra",
		"compiler/selfhostrt/actors_sysv.tetra",
		"compiler/selfhostrt/actors_win64.tetra",
	}
	required := append(actorRuntimeSymbols(), timeRuntimeSymbols()...)
	var missing []string
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			missing = append(missing, rel+": "+err.Error())
			continue
		}
		text := string(raw)
		for _, symbol := range required {
			if !strings.Contains(text, "@export("+strconv.Quote(symbol)+")") {
				missing = append(missing, rel+":"+symbol)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return failCheck("runtime exports", strings.Join(missing, ", "))
	}
	return passCheck(
		"runtime exports",
		fmt.Sprintf("%d files, %d symbols", len(paths), len(required)),
	)
}

func targetMetadataCheck() doctorCheck {
	entries := buildTargetReportEntries()
	seen := map[string]bool{}
	buildOnlyCount := 0
	for _, entry := range entries {
		if seen[entry.Triple] {
			return failCheck("target metadata", "duplicate target "+entry.Triple)
		}
		seen[entry.Triple] = true
		tgt, err := ctarget.Parse(entry.Triple)
		if err != nil {
			return failCheck("target metadata", err.Error())
		}
		if entry.OS != tgt.OS.String() || entry.Arch != tgt.Arch.String() ||
			entry.ABI != tgt.ABI.String() ||
			entry.DataModel != tgt.DataModel.String() ||
			entry.Format != tgt.Format.String() {
			return failCheck("target metadata", fmt.Sprintf("%s metadata mismatch", entry.Triple))
		}
		if entry.PointerWidthBits != tgt.PointerWidthBits ||
			entry.RegisterWidthBits != tgt.RegisterWidthBits ||
			entry.NativeIntWidthBits != tgt.NativeIntWidthBits {
			return failCheck(
				"target metadata",
				fmt.Sprintf("%s width metadata mismatch", entry.Triple),
			)
		}
		if entry.Endian != tgt.Endian.String() ||
			entry.StackAlignmentBytes != tgt.StackAlignmentBytes ||
			entry.MaxAtomicWidthBits != tgt.MaxAtomicWidthBits {
			return failCheck(
				"target metadata",
				fmt.Sprintf("%s layout metadata mismatch", entry.Triple),
			)
		}
		if !sameDoctorInts(entry.AtomicWidthBits, tgt.AtomicWidthBits()) ||
			entry.AtomicPointerWidthBits != atomicPointerWidthBits(tgt) {
			return failCheck(
				"target metadata",
				fmt.Sprintf("%s atomic metadata mismatch", entry.Triple),
			)
		}
		if entry.UnsupportedReason != tgt.UnsupportedReason {
			return failCheck(
				"target metadata",
				fmt.Sprintf(
					"%s unsupported_reason got %q want %q",
					entry.Triple,
					entry.UnsupportedReason,
					tgt.UnsupportedReason,
				),
			)
		}
		if entry.ExeExt != tgt.ExeExt {
			return failCheck(
				"target metadata",
				fmt.Sprintf("%s exe_ext got %q want %q", entry.Triple, entry.ExeExt, tgt.ExeExt),
			)
		}
		buildOnly := ctarget.IsBuildOnlyTarget(entry.Triple)
		if entry.BuildOnly != buildOnly {
			return failCheck(
				"target metadata",
				fmt.Sprintf(
					"%s build_only got %v want %v",
					entry.Triple,
					entry.BuildOnly,
					buildOnly,
				),
			)
		}
		if err := validateDoctorTargetRunMetadata(entry, tgt, buildOnly); err != nil {
			return failCheck("target metadata", err.Error())
		}
		if buildOnly {
			buildOnlyCount++
		}
	}
	wantCount := len(
		ctarget.SupportedTriples(),
	) + len(
		ctarget.BuildOnlyTriples(),
	) + len(
		ctarget.PlannedTriples(),
	)
	if len(entries) != wantCount {
		return failCheck(
			"target metadata",
			fmt.Sprintf("got %d targets want %d", len(entries), wantCount),
		)
	}
	return passCheck(
		"target metadata",
		fmt.Sprintf("%d targets, %d build-only", len(entries), buildOnlyCount),
	)
}

func validateDoctorTargetRunMetadata(
	entry targetReportEntry,
	tgt ctarget.Target,
	buildOnly bool,
) error {
	if entry.RunMode != tgt.RunMode.String() {
		return fmt.Errorf(
			"%s run_mode got %q want %q",
			entry.Triple,
			entry.RunMode,
			tgt.RunMode.String(),
		)
	}
	switch tgt.RunMode {
	case ctarget.RunModeHostNative:
		if buildOnly {
			return fmt.Errorf("%s host_native run mode cannot be build-only", entry.Triple)
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("%s run_runner got %q want empty", entry.Triple, entry.RunRunner)
		}
		if !entry.RunSupported && entry.RunUnsupportedReason == "" {
			return fmt.Errorf(
				"%s run_unsupported_reason is required when run_supported is false",
				entry.Triple,
			)
		}
	case ctarget.RunModeHostProbed:
		if !buildOnly {
			return fmt.Errorf("%s host_probed run mode must be build-only", entry.Triple)
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("%s run_runner got %q want empty", entry.Triple, entry.RunRunner)
		}
		if entry.RunSupported {
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf(
					"%s run_unsupported_reason got %q want empty when host probe is supported",
					entry.Triple,
					entry.RunUnsupportedReason,
				)
			}
			return nil
		}
		if !strings.Contains(entry.RunUnsupportedReason, "no host fallback") {
			return fmt.Errorf(
				"%s run_unsupported_reason must explain host probe failure and no fallback",
				entry.Triple,
			)
		}
	case ctarget.RunModeWASIRunner:
		if entry.Triple != "wasm32-wasi" || buildOnly {
			return fmt.Errorf(
				"%s wasi_runner mode is only valid for wasm32-wasi supported target",
				entry.Triple,
			)
		}
		if entry.RunSupported {
			if entry.RunRunner != "wasmtime" && entry.RunRunner != "node-wasi" {
				return fmt.Errorf(
					"%s run_runner got %q want wasmtime or node-wasi",
					entry.Triple,
					entry.RunRunner,
				)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf(
					"%s run_unsupported_reason got %q want empty when runner is available",
					entry.Triple,
					entry.RunUnsupportedReason,
				)
			}
			return nil
		}
		if entry.RunRunner != "" {
			return fmt.Errorf(
				"%s run_runner got %q want empty when runner is unavailable",
				entry.Triple,
				entry.RunRunner,
			)
		}
		if !strings.Contains(entry.RunUnsupportedReason, "missing WASI runner") {
			return fmt.Errorf(
				"%s run_unsupported_reason must explain missing WASI runner",
				entry.Triple,
			)
		}
	case ctarget.RunModeWebRunner:
		if entry.Triple != "wasm32-web" || buildOnly {
			return fmt.Errorf(
				"%s web_runner mode is only valid for wasm32-web supported target",
				entry.Triple,
			)
		}
		if entry.RunSupported {
			if entry.RunRunner == "" {
				return fmt.Errorf(
					"%s run_runner is required when web runner is available",
					entry.Triple,
				)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf(
					"%s run_unsupported_reason got %q want empty when web runner is available",
					entry.Triple,
					entry.RunUnsupportedReason,
				)
			}
			return nil
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("%s run_runner got %q want empty", entry.Triple, entry.RunRunner)
		}
		if !strings.Contains(entry.RunUnsupportedReason, "web runner unavailable") &&
			!strings.Contains(entry.RunUnsupportedReason, "browser runner unavailable") &&
			!strings.Contains(entry.RunUnsupportedReason, "missing web runtime runner") &&
			!strings.Contains(entry.RunUnsupportedReason, "missing web node helper") {
			return fmt.Errorf(
				"%s run_unsupported_reason must explain missing web runner",
				entry.Triple,
			)
		}
	case ctarget.RunModeUnsupported:
		if !buildOnly {
			return fmt.Errorf("%s unsupported run mode must be build-only", entry.Triple)
		}
		if entry.RunSupported {
			return fmt.Errorf("%s run_supported got true for unsupported run mode", entry.Triple)
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("%s run_runner got %q want empty", entry.Triple, entry.RunRunner)
		}
		if entry.RunUnsupportedReason == "" {
			return fmt.Errorf(
				"%s run_unsupported_reason is required for unsupported run mode",
				entry.Triple,
			)
		}
	default:
		return fmt.Errorf("%s has unsupported run_mode %q", entry.Triple, tgt.RunMode.String())
	}
	return nil
}

func sameDoctorInts(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toolingCommandsCheck() doctorCheck {
	commands := []string{
		"check",
		"build",
		"run",
		"fmt",
		"test",
		"doc",
		"interface",
		"smoke",
		"targets",
		"formats",
		"doctor",
		"actor-net",
		"project",
		"new",
		"lsp",
		"eco",
		"clean",
		"version",
	}
	if len(commands) == 0 {
		return failCheck("tooling commands", "no commands registered")
	}
	return passCheck("tooling commands", strings.Join(commands, ", "))
}

func actorRuntimeSymbols() []string {
	return []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	}
}

func timeRuntimeSymbols() []string {
	return []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func sortedDoctorStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func pathCheck(root string, rel string) doctorCheck {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		return failCheck(rel, err.Error())
	}
	return passCheck(rel, "found")
}

func passCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "pass", Detail: detail}
}

func failCheck(name string, detail string) doctorCheck {
	return doctorCheck{Name: name, Status: "fail", Detail: detail}
}

func doctorStatus(checks []doctorCheck) string {
	for _, check := range checks {
		if check.Status != "pass" {
			return "fail"
		}
	}
	return "pass"
}

// ---- interface.go ----

func runInterface(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("interface", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output .t4i path; stdout when empty")
	checkMode := fs.Bool("check", false, "check that the .t4i public API hash matches the source")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "interface requires exactly one input path")
		return 2
	}
	inputPath := fs.Arg(0)
	if *checkMode {
		path := *outPath
		if path == "" {
			path = compiler.InterfaceOutputPath(inputPath)
		}
		src, err := os.ReadFile(inputPath)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		iface, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if err := compiler.ValidateInterfaceAgainstSource(src, iface, inputPath); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Interface current: %s\n", path)
		return 0
	}
	docs, err := compiler.GenerateInterfaceFile(inputPath)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote interface: %s\n", *outPath)
	return 0
}

// ---- metadata.go ----

type targetsReport struct {
	Supported []string            `json:"supported"`
	BuildOnly []string            `json:"build_only"`
	Planned   []string            `json:"planned"`
	Targets   []targetReportEntry `json:"targets"`
}

type targetReportEntry struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	BuildOnly                bool     `json:"build_only"`
	RunMode                  string   `json:"run_mode"`
	RunRunner                string   `json:"run_runner,omitempty"`
	RunSupported             bool     `json:"run_supported"`
	RunUnsupportedReason     string   `json:"run_unsupported_reason,omitempty"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status"`
	UIRuntimeEvidence        string   `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits         int      `json:"pointer_width_bits"`
	RegisterWidthBits        int      `json:"register_width_bits"`
	NativeIntWidthBits       int      `json:"native_int_width_bits"`
	Endian                   string   `json:"endian"`
	StackAlignmentBytes      int      `json:"stack_alignment_bytes"`
	MaxAtomicWidthBits       int      `json:"max_atomic_width_bits"`
	AtomicWidthBits          []int    `json:"atomic_width_bits"`
	AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits"`
	UnsupportedReason        string   `json:"unsupported_reason,omitempty"`
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StdlibStatus             string   `json:"stdlib_status,omitempty"`
	FFIStatus                string   `json:"ffi_status,omitempty"`
	MemoryBuild              string   `json:"memory_build"`
	MemoryLower              string   `json:"memory_lower"`
	MemoryRun                string   `json:"memory_run"`
	MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics"`
	MemoryRegionLowering     string   `json:"memory_region_lowering"`
	MemoryAlignmentSemantics string   `json:"memory_alignment_semantics"`
	MemoryClaimLevel         string   `json:"memory_claim_level"`
	RunnerProbeCommand       string   `json:"runner_probe_command,omitempty"`
	ReleaseGate              string   `json:"release_gate,omitempty"`
	EvidenceArtifacts        []string `json:"evidence_artifacts,omitempty"`
	SyscallInstruction       string   `json:"syscall_instruction,omitempty"`
	SyscallNumbering         string   `json:"syscall_numbering,omitempty"`
	SyscallArgRegisters      []string `json:"syscall_arg_registers,omitempty"`
	SyscallErrorRange        string   `json:"syscall_error_range,omitempty"`
	SupportsDebugInfo        bool     `json:"supports_debug_info"`
	SupportsReleaseOptimize  bool     `json:"supports_release_optimize"`
}

type formatsReport struct {
	Formats []compiler.FormatInfo `json:"formats"`
}

type featuresReport struct {
	Schema   string                 `json:"schema"`
	Version  string                 `json:"version"`
	Features []compiler.FeatureInfo `json:"features"`
}

func runTargets(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("targets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "targets does not accept positional arguments")
		return 2
	}
	report := targetsReport{
		Supported: ctarget.SupportedTriples(),
		BuildOnly: ctarget.BuildOnlyTriples(),
		Planned:   ctarget.PlannedTriples(),
		Targets:   buildTargetReportEntries(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "Supported targets:")
		for _, triple := range report.Supported {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Build-only targets:")
		for _, triple := range report.BuildOnly {
			fmt.Fprintf(stdout, "  %s\n", describeTargetForText(triple))
		}
		fmt.Fprintln(stdout, "Planned targets:")
		for _, triple := range report.Planned {
			fmt.Fprintf(stdout, "  %s\n", triple)
		}
		return 0
	case "json", "toon":
		if err := outputformat.WriteStructured(stdout, *format, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runFeatures(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("features", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "features does not accept positional arguments")
		return 2
	}
	report := featuresReport{
		Schema:   "tetra.features.v1",
		Version:  compiler.Version(),
		Features: compiler.FeatureRegistry(),
	}
	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "Tetra features (%s):\n", report.Version)
		for _, feature := range report.Features {
			fmt.Fprintf(stdout, "  %s [%s] - %s\n", feature.ID, feature.Status, feature.Name)
		}
		return 0
	case "json", "toon":
		if err := outputformat.WriteStructured(stdout, *format, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runFormats(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("formats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "formats does not accept positional arguments")
		return 2
	}
	report := formatsReport{Formats: compiler.T4Formats()}
	switch *format {
	case "text", "":
		fmt.Fprintln(stdout, "T4 formats:")
		for _, item := range report.Formats {
			suffix := item.Extension
			if suffix == "" {
				suffix = item.FileName
			}
			markers := []string{item.Role}
			if item.Primary {
				markers = append(markers, "primary")
			}
			if item.Legacy {
				markers = append(markers, "legacy")
			}
			fmt.Fprintf(stdout, "  %s - %s (%s)\n", suffix, item.Name, strings.Join(markers, ", "))
		}
		return 0
	case "json", "toon":
		if err := outputformat.WriteStructured(stdout, *format, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func buildTargetReportEntries() []targetReportEntry {
	host, hostOK := hostTarget()
	triples := append([]string{}, ctarget.SupportedTriples()...)
	triples = append(triples, ctarget.BuildOnlyTriples()...)
	triples = append(triples, ctarget.PlannedTriples()...)
	out := make([]targetReportEntry, 0, len(triples))
	for _, triple := range triples {
		tgt, err := ctarget.Parse(triple)
		if err != nil {
			continue
		}
		buildOnly := ctarget.IsBuildOnlyTarget(tgt.Triple)
		runSupported, runRunner, runUnsupportedReason := targetRunSupport(tgt, host, hostOK)
		out = append(out, targetReportEntry{
			Triple:                   tgt.Triple,
			Status:                   tgt.Status.String(),
			OS:                       tgt.OS.String(),
			Arch:                     tgt.Arch.String(),
			ABI:                      tgt.ABI.String(),
			DataModel:                tgt.DataModel.String(),
			Format:                   tgt.Format.String(),
			ExeExt:                   tgt.ExeExt,
			BuildOnly:                buildOnly,
			RunMode:                  tgt.RunMode.String(),
			RunRunner:                runRunner,
			RunSupported:             runSupported,
			RunUnsupportedReason:     runUnsupportedReason,
			UIRuntimeContract:        ctarget.UIRuntimeContract(tgt.Triple),
			UIRuntimeStatus:          ctarget.UIRuntimeStatus(tgt.Triple),
			UIRuntimeEvidence:        ctarget.UIRuntimeEvidence(tgt.Triple),
			PointerWidthBits:         tgt.PointerWidthBits,
			RegisterWidthBits:        tgt.RegisterWidthBits,
			NativeIntWidthBits:       tgt.NativeIntWidthBits,
			Endian:                   tgt.Endian.String(),
			StackAlignmentBytes:      tgt.StackAlignmentBytes,
			MaxAtomicWidthBits:       tgt.MaxAtomicWidthBits,
			AtomicWidthBits:          tgt.AtomicWidthBits(),
			AtomicPointerWidthBits:   atomicPointerWidthBits(tgt),
			UnsupportedReason:        tgt.UnsupportedReason,
			RuntimeStatus:            tgt.RuntimeStatus,
			StdlibStatus:             tgt.StdlibStatus,
			FFIStatus:                tgt.FFIStatus,
			MemoryBuild:              tgt.MemoryBuild,
			MemoryLower:              tgt.MemoryLower,
			MemoryRun:                tgt.MemoryRun,
			MemoryRawDiagnostics:     tgt.MemoryRawDiagnostics,
			MemoryRegionLowering:     tgt.MemoryRegionLowering,
			MemoryAlignmentSemantics: tgt.MemoryAlignmentSemantics,
			MemoryClaimLevel:         tgt.MemoryClaimLevel,
			RunnerProbeCommand:       tgt.RunnerProbeCommand,
			ReleaseGate:              tgt.ReleaseGate,
			EvidenceArtifacts:        append([]string(nil), tgt.EvidenceArtifacts...),
			SyscallInstruction:       tgt.SyscallInstruction,
			SyscallNumbering:         tgt.SyscallNumbering,
			SyscallArgRegisters:      append([]string(nil), tgt.SyscallArgRegisters...),
			SyscallErrorRange:        tgt.SyscallErrorRange,
			SupportsDebugInfo:        tgt.SupportsDebugInfo,
			SupportsReleaseOptimize:  tgt.SupportsReleaseOptimize,
		})
	}
	return out
}

func atomicPointerWidthBits(tgt ctarget.Target) int {
	layout, err := tgt.AtomicPointerLayout()
	if err != nil {
		return 0
	}
	return layout.WidthBits
}

func targetRunSupport(tgt ctarget.Target, host string, hostOK bool) (bool, string, string) {
	switch tgt.RunMode {
	case ctarget.RunModeHostNative:
		if hostOK && host == tgt.Triple {
			return true, "", ""
		}
		return false, "", fmt.Sprintf(
			"%s cannot run on host %s/%s",
			tgt.Triple,
			runtime.GOOS,
			runtime.GOARCH,
		)
	case ctarget.RunModeHostProbed:
		if !ctarget.IsBuildOnlyTarget(tgt.Triple) {
			return false, "", fmt.Sprintf(
				"%s host_probed run mode is only valid for build-only native targets",
				tgt.Triple,
			)
		}
		if canRunBuildOnlyNativeTargetOnHost(tgt) {
			return true, "", ""
		}
		return false, "", buildOnlyNativeRunUnsupportedReason(tgt)
	case ctarget.RunModeWASIRunner:
		runner, err := discoverWASIRunner("")
		if err != nil {
			return false, "", err.Error()
		}
		return true, runner.Name, ""
	case ctarget.RunModeWebRunner:
		runner, err := discoverWebRunner()
		if err != nil {
			return false, "", err.Error()
		}
		return true, runner, ""
	case ctarget.RunModeUnsupported:
		if tgt.UnsupportedReason != "" {
			return false, "", tgt.UnsupportedReason
		}
		return false, "", fmt.Sprintf("%s has unsupported runtime mode", tgt.Triple)
	default:
		return false, "", fmt.Sprintf(
			"%s has unknown runtime mode %s",
			tgt.Triple,
			tgt.RunMode.String(),
		)
	}
}

func describeTargetForText(triple string) string {
	tgt, err := ctarget.Parse(triple)
	if err != nil {
		return triple
	}
	parts := []string{
		triple,
		"os=" + tgt.OS.String(),
		"arch=" + tgt.Arch.String(),
		"abi=" + tgt.ABI.String(),
		"data_model=" + tgt.DataModel.String(),
		"format=" + tgt.Format.String(),
		fmt.Sprintf("ptr=%d", tgt.PointerWidthBits),
		fmt.Sprintf("reg=%d", tgt.RegisterWidthBits),
		fmt.Sprintf("native_int=%d", tgt.NativeIntWidthBits),
		"endian=" + tgt.Endian.String(),
	}
	if tgt.ExeExt != "" {
		parts = append(parts, "exe_ext="+tgt.ExeExt)
	}
	if ctarget.IsBuildOnlyTarget(triple) {
		parts = append(parts, "build-only")
	}
	if tgt.RunMode != ctarget.RunModeUnknown {
		parts = append(parts, "run_mode="+tgt.RunMode.String())
	}
	if tgt.UnsupportedReason != "" {
		parts = append(parts, "unsupported_reason="+tgt.UnsupportedReason)
	}
	return strings.Join(parts, " ")
}

// ---- new_app.go ----

func runNew(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new {app|surface-app} [options] <NameOrPath>")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "new requires a template")
		return 2
	}
	switch args[0] {
	case "app":
		return runNewAppArgs(args[1:], stdout, stderr)
	case "surface-app":
		return runNewSurfaceAppArgs(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown new template %q\n", args[0])
		return 2
	}
}

type newAppOptions struct {
	WriteLock bool
}

func runNewAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		return 0
	}
	var path string
	var opt newAppOptions
	for _, arg := range args {
		switch arg {
		case "--lock":
			opt.WriteLock = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
		return 2
	}
	return runNewApp(path, opt, stdout, stderr)
}

func runNewApp(path string, opt newAppOptions, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new app requires a name or path")
		return 2
	}
	targetDir := filepath.Clean(filepath.FromSlash(path))
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", targetDir)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	name := capsuleNameFromPath(targetDir)
	if name == "" {
		fmt.Fprintln(stderr, "new app requires a valid app name")
		return 2
	}
	target := defaultTarget()
	files := map[string]string{
		"Capsule.t4": fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://apps/%s"
    version "0.1.0"
    entry "src/main.t4"
    source "src"
    source "tests"
    target "%s"
    permission "io"
`, name, capsuleSlug(name), target),
		"src/main.t4": `func main() -> Int:
    return 0
`,
		"tests/main_test.t4": `test "main returns success":
    expect 40 + 2 == 42
`,
		"README.md": fmt.Sprintf(`# %s

Run:

`+"```bash"+`
tetra check .
tetra build .
tetra run .
tetra test .
`+"```"+`
`, name),
	}
	for rel, content := range files {
		full := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Created app: %s\n", targetDir)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, compiler.SemanticLockFileName)
		if err := buildCapsuleArtifacts(filepath.Join(
			targetDir,
			compiler.CapsuleFileName,
		), capsuleArtifactBuildOptions{
			LockPath: lockPath,
			Jobs:     1,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Created lock: %s\n", lockPath)
	}
	return 0
}

func capsuleNameFromPath(path string) string {
	name := filepath.Base(filepath.Clean(path))
	var b strings.Builder
	capitalizeNext := true
	for _, r := range name {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if b.Len() == 0 && r >= '0' && r <= '9' {
				b.WriteByte('T')
			}
			if capitalizeNext && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			b.WriteRune(r)
			capitalizeNext = false
			continue
		}
		capitalizeNext = true
	}
	return b.String()
}

func capsuleSlug(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a')
			lastDash = false
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// ---- new_surface_app.go ----

const surfaceProjectTemplateModel = "surface-project-template-v1"

var surfaceAppTemplateKinds = []string{
	"command-palette",
	"settings",
	"dashboard",
	"editor-shell",
	"studio-shell",
	"multi-window-notes",
	"web-canvas",
}

type newSurfaceAppOptions struct {
	Template  string
	WriteLock bool
}

type surfaceAppTemplateSpec struct {
	Kind        string
	TitleHash   int
	AccentAlpha int
	Recipes     []string
	Actions     []int
	AppShell    bool
	WebCanvas   bool
}

func runNewSurfaceAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>")
		fmt.Fprintf(stdout, "templates: %s\n", strings.Join(surfaceAppTemplateKinds, ", "))
		return 0
	}
	var path string
	opt := newSurfaceAppOptions{Template: "command-palette"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--lock":
			opt.WriteLock = true
		case "--template":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "new surface-app --template requires a value")
				return 2
			}
			opt.Template = args[i+1]
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new surface-app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(
					stderr,
					"usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>",
				)
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new surface-app [--template KIND] [--lock] <NameOrPath>")
		return 2
	}
	return runNewSurfaceApp(path, opt, stdout, stderr)
}

func runNewSurfaceApp(
	path string,
	opt newSurfaceAppOptions,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new surface-app requires a name or path")
		return 2
	}
	spec, ok := surfaceAppTemplateByKind(opt.Template)
	if !ok {
		fmt.Fprintf(stderr, "unknown surface app template %q\n", opt.Template)
		return 2
	}
	targetDir := filepath.Clean(filepath.FromSlash(path))
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", targetDir)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	name := capsuleNameFromPath(targetDir)
	if name == "" {
		fmt.Fprintln(stderr, "new surface-app requires a valid app name")
		return 2
	}
	target := defaultTarget()
	files := map[string]string{
		"Capsule.t4":            surfaceAppCapsule(name, target),
		"src/main.tetra":        surfaceAppSource(spec),
		"surface-template.json": surfaceAppTemplateMetadata(name, spec, target),
		"README.md":             surfaceAppReadme(name, spec),
		"tests/main_test.tetra": surfaceAppTemplateTestSource(),
		"design/recipes.tetra":  surfaceAppDesignRecipesSource(spec),
		"design/tokens.tetra":   surfaceAppDesignTokensSource(),
	}
	for rel, content := range files {
		full := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Created Surface app: %s\n", targetDir)
	fmt.Fprintf(stdout, "Surface template: %s\n", spec.Kind)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, "Tetra.lock")
		if err := buildCapsuleArtifacts(filepath.Join(
			targetDir,
			"Capsule.t4",
		), capsuleArtifactBuildOptions{
			LockPath: lockPath,
			Jobs:     1,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Created lock: %s\n", lockPath)
	}
	return 0
}

func surfaceAppTemplateByKind(kind string) (surfaceAppTemplateSpec, bool) {
	switch strings.TrimSpace(kind) {
	case "command-palette":
		return surfaceAppTemplateSpec{
			Kind:        "command-palette",
			TitleHash:   16,
			AccentAlpha: 44,
			Recipes: []string{
				"recipe_region_panel",
				"recipe_field_text",
				"recipe_command_item",
				"recipe_control_action",
			},
			Actions: []int{301, 302, 303},
		}, true
	case "settings":
		return surfaceAppTemplateSpec{
			Kind:        "settings",
			TitleHash:   8,
			AccentAlpha: 30,
			Recipes: []string{
				"recipe_form_field",
				"recipe_field_text",
				"recipe_tab_item",
				"recipe_control_action",
			},
			Actions: []int{501, 502, 503},
		}, true
	case "dashboard":
		return surfaceAppTemplateSpec{
			Kind:        "dashboard",
			TitleHash:   17,
			AccentAlpha: 36,
			Recipes: []string{
				"recipe_region_panel",
				"recipe_metric_tile",
				"recipe_list_row",
				"recipe_toast_notification",
			},
			Actions: []int{401, 402, 403},
		}, true
	case "editor-shell":
		return surfaceAppTemplateSpec{
			Kind:        "editor-shell",
			TitleHash:   12,
			AccentAlpha: 34,
			Recipes: []string{
				"recipe_nav_item",
				"recipe_tab_item",
				"recipe_command_item",
				"recipe_region_panel",
			},
			Actions: []int{601, 602, 603},
		}, true
	case "studio-shell":
		return surfaceAppTemplateSpec{
			Kind:        "studio-shell",
			TitleHash:   18,
			AccentAlpha: 42,
			Recipes: []string{
				"recipe_app_shell",
				"recipe_toolbar",
				"recipe_split_pane",
				"recipe_status_bar",
			},
			Actions:  []int{901, 902, 903},
			AppShell: true,
		}, true
	case "multi-window-notes":
		return surfaceAppTemplateSpec{
			Kind:        "multi-window-notes",
			TitleHash:   9,
			AccentAlpha: 38,
			Recipes: []string{
				"recipe_region_panel",
				"recipe_list_row",
				"recipe_field_text",
				"recipe_control_action",
			},
			Actions:  []int{701, 702, 703},
			AppShell: true,
		}, true
	case "web-canvas":
		return surfaceAppTemplateSpec{
			Kind:        "web-canvas",
			TitleHash:   21,
			AccentAlpha: 40,
			Recipes: []string{
				"recipe_region_panel",
				"recipe_metric_tile",
				"recipe_command_item",
				"recipe_field_text",
			},
			Actions:   []int{801, 802, 803},
			WebCanvas: true,
		}, true
	default:
		return surfaceAppTemplateSpec{}, false
	}
}

func surfaceAppCapsule(name string, target string) string {
	return fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://surface-apps/%s"
    version "0.1.0"
    entry "src/main.tetra"
    source "src"
    source "design"
    target "%s"
    target "wasm32-web"
    permission "io"
`, name, capsuleSlug(name), target)
}

func surfaceAppSource(spec surfaceAppTemplateSpec) string {
	joinLine := func(parts ...string) string {
		return strings.Join(parts, "")
	}
	imports := `import lib.core.surface as surface
import lib.core.block as block
import lib.core.morph as morph
`
	if spec.AppShell {
		imports += "import lib.core.surface_app_shell as shell\n"
	}
	shellLifecycleLine := joinLine(
		"    let lifecycle: shell.ShellFeature = ",
		"shell.target_evidenced_feature(shell.feature_window_lifecycle())",
	)
	shellReadyLine := joinLine(
		"    if open_main == 1 && open_detail == 1 && ",
		"resize_main == 1 && cursor == 1 && ",
		"shell.multi_window_ready(main_win, detail_win) && ",
		"shell.feature_is_honest(menu) && ",
		"shell.feature_is_honest(lifecycle) && ",
		"shell.feature_is_honest(multi) && ",
		"shell.feature_is_honest(blocked):",
	)
	shellFunc := `func template_shell_score() -> Int:
    return 1
`
	if spec.AppShell {
		shellFunc = fmt.Sprintf(`func template_shell_score() -> Int:
    var main_win: shell.ShellWindow = shell.window(1, 5, 560, 420, 1000)
    var detail_win: shell.ShellWindow = shell.window(2, 9, 320, 240, 1000)
    let open_main: Int = shell.open_window(main_win)
    let open_detail: Int = shell.open_window(detail_win)
    let resize_main: Int = shell.resize_window(main_win, 720, 540)
    let cursor: Int = shell.set_cursor(main_win, shell.cursor_text())
    let menu: shell.ShellFeature = shell.scoped_adapter_feature(shell.feature_app_menu())
%s
    let multi: shell.ShellFeature = shell.target_evidenced_feature(shell.feature_multi_window())
    let blocked: shell.ShellFeature = shell.blocked_pass_feature(shell.feature_notification())
%s
        return 1
    return 0
`, shellLifecycleLine, shellReadyLine)
	}
	targetFunc := `func template_target_score() -> Int:
    return 1
`
	if spec.WebCanvas {
		targetFunc = `func template_target_score() -> Int:
    let size: surface.Size = surface.Size(w: 640, h: 360)
    if size.w == 640 && size.h == 360:
        return 1
    return 0
`
	}
	rootPaintLine := fmt.Sprintf(
		joinLine(
			"    let paint: block.PaintSpec = block.paint_stack2(",
			"block.paint_layer_fill_radius(morph.theme_dark(), 0), ",
			"block.paint_layer_overlay(morph.accent(), %d, 0))",
		),
		spec.AccentAlpha,
	)
	rootPropsLine := fmt.Sprintf(
		joinLine(
			"    let props: block.BlockProps = block.props(",
			"block.layout_overlay(rect, 0), paint, ",
			"block.text_none(), block.image_none(), ",
			"block.input_none(), block.event_none(), ",
			"block.state_base(), block.motion_none(), ",
			"block.accessibility_region(%d), block.asset_none())",
		),
		spec.TitleHash,
	)
	recipeGuardLine := joinLine(
		"    if morph.capsule_valid(capsule) && ",
		"morph.expansion_valid(first_expansion) && ",
		"morph.expansion_valid(second_expansion) && ",
		"morph.expansion_valid(third_expansion) && ",
		"morph.expansion_valid(fourth_expansion) && ",
		"morph.recipe_expands_to_block(first_recipe) && ",
		"morph.recipe_expands_to_block(second_recipe) && ",
		"morph.recipe_expands_to_block(third_recipe) && ",
		"morph.recipe_expands_to_block(fourth_recipe):",
	)
	recipeChecksumLine := joinLine(
		"        return capsule.token_graph_hash + ",
		"first_recipe.name_hash + second_recipe.name_hash + ",
		"third_recipe.name_hash + fourth_recipe.name_hash",
	)
	panelLine := fmt.Sprintf(
		joinLine(
			"    let panel: Int = block.tree_add_child(",
			"tree, block.id(1), ",
			"morph.expand_region_panel(2, 1, panel_rect, %d), ",
			"panel_rect)",
		),
		spec.TitleHash,
	)
	labelLine := joinLine(
		"    let label: Int = block.tree_add_child(",
		"tree, block.id(2), ",
		"morph.expand_label(3, 2, 4, label_rect, 6), ",
		"label_rect)",
	)
	fieldLine := joinLine(
		"    let field: Int = block.tree_add_child(",
		"tree, block.id(2), ",
		"morph.expand_field_text(4, 2, 3, field_rect, 18, 8), ",
		"field_rect)",
	)
	primaryLine := fmt.Sprintf(
		joinLine(
			"    let primary: Int = block.tree_add_child(",
			"tree, block.id(2), ",
			"morph.expand_control_action(5, 2, primary_rect, 13, %d, true), ",
			"primary_rect)",
		),
		spec.Actions[0],
	)
	secondaryLine := fmt.Sprintf(
		joinLine(
			"    let secondary: Int = block.tree_add_child(",
			"tree, block.id(2), ",
			"morph.expand_control_action(6, 2, secondary_rect, 17, %d, false), ",
			"secondary_rect)",
		),
		spec.Actions[1],
	)
	tertiaryLine := fmt.Sprintf(
		joinLine(
			"    let tertiary: Int = block.tree_add_child(",
			"tree, block.id(2), ",
			"morph.expand_control_action(7, 2, tertiary_rect, 12, %d, false), ",
			"tertiary_rect)",
		),
		spec.Actions[2],
	)
	appLine := joinLine(
		"    let app: SurfaceTemplateApp = SurfaceTemplateApp(",
		"template_hash: template_recipe_checksum(), recipe_count: 4, ",
		"block_count: block.tree_len(tree), ",
		"shell_score: template_shell_score(), ",
		"target_score: template_target_score())",
	)
	successGuardLine := joinLine(
		"    if root == 1 && panel == 2 && label == 3 && field == 4 && ",
		"primary == 5 && secondary == 6 && tertiary == 7 && ",
		"valid == block.tree_error_ok() && focus0 == 4 && focus1 == 5 && ",
		"a11y0 == 1 && app.template_hash > 0 && app.recipe_count == 4 && ",
		"app.block_count == 7 && app.shell_score == 1 && app.target_score == 1 && ",
		"morph.capsule_self_check() && ",
		"morph.accessibility_projection_ok(3, 4, 6) && ",
		"morph.memory_budget_ok(app.block_count, 3, 64 * 64 * 4):",
	)
	return fmt.Sprintf(`// Surface %s template authored through Block/Morph recipes.
module main

%s
struct SurfaceTemplateApp:
    template_hash: Int
    recipe_count: Int
    block_count: Int
    shell_score: Int
    target_score: Int

func root_block(rect: surface.Rect) -> block.Block:
%s
%s
    return block.make(block.id(1), block.id_none(), props)

%s
%s
func template_recipe_checksum() -> Int:
    let capsule: morph.Capsule = morph.capsule_default()
    let first_recipe: morph.Recipe = morph.%s()
    let second_recipe: morph.Recipe = morph.%s()
    let third_recipe: morph.Recipe = morph.%s()
    let fourth_recipe: morph.Recipe = morph.%s()
    let first_expansion: morph.RecipeExpansion = morph.recipe_expansion(first_recipe, block.id(2))
    let second_expansion: morph.RecipeExpansion = morph.recipe_expansion(second_recipe, block.id(4))
    let third_expansion: morph.RecipeExpansion = morph.recipe_expansion(third_recipe, block.id(5))
    let fourth_expansion: morph.RecipeExpansion = morph.recipe_expansion(fourth_recipe, block.id(6))
%s
%s
    return 0

func main() -> Int
uses alloc, mem:
    let root_rect: surface.Rect = surface.Rect(x: 0, y: 0, w: 420, h: 280)
    let panel_rect: surface.Rect = surface.Rect(x: 20, y: 20, w: 380, h: 240)
    let label_rect: surface.Rect = surface.Rect(x: 32, y: 24, w: 160, h: 18)
    let field_rect: surface.Rect = surface.Rect(x: 32, y: 56, w: 344, h: 42)
    let primary_rect: surface.Rect = surface.Rect(x: 32, y: 112, w: 344, h: 44)
    let secondary_rect: surface.Rect = surface.Rect(x: 32, y: 164, w: 344, h: 44)
    let tertiary_rect: surface.Rect = surface.Rect(x: 32, y: 216, w: 344, h: 28)
    var tree: block.BlockTree = block.tree_init(8)
    let root: Int = block.tree_add_root(tree, root_block(root_rect), root_rect)
%s
%s
%s
%s
%s
%s
%s
    let valid: Int = block.tree_validate(tree)
    let focus0: Int = block.id_value(block.focus_order_at(tree, 0))
    let focus1: Int = block.id_value(block.focus_order_at(tree, 1))
    let a11y0: Int = block.id_value(block.tree_accessibility_order_at(tree, 0))
%s
        return 0
    return 1
`,
		spec.Kind,
		imports,
		rootPaintLine,
		rootPropsLine,
		shellFunc,
		targetFunc,
		spec.Recipes[0],
		spec.Recipes[1],
		spec.Recipes[2],
		spec.Recipes[3],
		recipeGuardLine,
		recipeChecksumLine,
		panelLine,
		labelLine,
		fieldLine,
		primaryLine,
		secondaryLine,
		tertiaryLine,
		appLine,
		successGuardLine,
	)
}

func surfaceAppTemplateMetadata(name string, spec surfaceAppTemplateSpec, target string) string {
	imports := []string{"lib.core.surface", "lib.core.block", "lib.core.morph"}
	if spec.AppShell {
		imports = append(imports, "lib.core.surface_app_shell")
	}
	return fmt.Sprintf(`{
  "schema": "tetra.surface.project-template.v1",
  "model": "%s",
  "template": "%s",
  "app": "%s",
  "release_scope": "surface-v1-linux-web",
  "entry": "src/main.tetra",
  "targets": ["%s", "wasm32-web"],
  "imports": [%s],
  "block_morph_only": true,
  "uses_app_shell": %t,
  "web_canvas": %t,
  "negative_guards": {
    "no_react_import": true,
    "no_electron_import": true,
    "no_dom_app_ui_tree": true,
    "no_css_runtime": true,
    "no_core_widgets": true,
    "no_platform_widgets": true,
    "no_user_js_app_logic": true
  }
}
`, surfaceProjectTemplateModel, spec.Kind, name, target, quotedJSONList(
		imports,
	), spec.AppShell, spec.WebCanvas)
}

func surfaceAppReadme(name string, spec surfaceAppTemplateSpec) string {
	return fmt.Sprintf(`# %s

Surface template: %s

This project is authored with `+"`Block`"+` and `+"`Morph`"+` recipes.

`+"```bash"+`
tetra check .
tetra build --target linux-x64 .
tetra run --target linux-x64 .
tetra build --target wasm32-web .
`+"```"+`
`, name, spec.Kind)
}

func surfaceAppTemplateTestSource() string {
	return `test "surface template math sanity":
    expect 40 + 2 == 42
`
}

func surfaceAppDesignRecipesSource(spec surfaceAppTemplateSpec) string {
	return fmt.Sprintf(`// Recipe catalog marker for the %s Surface template.
func template_recipe_count() -> Int:
    return 4
`, spec.Kind)
}

func surfaceAppDesignTokensSource() string {
	return `// Token catalog marker consumed by Surface template smoke evidence.
func template_token_count() -> Int:
    return 3
`
}

func quotedJSONList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}

// ---- surface_dev.go ----

type surfaceDevReport struct {
	Schema                 string                            `json:"schema"`
	Model                  string                            `json:"model"`
	ReleaseScope           string                            `json:"release_scope"`
	Command                string                            `json:"command"`
	Source                 string                            `json:"source"`
	Target                 string                            `json:"target"`
	Mode                   string                            `json:"mode"`
	ReloadSemantics        string                            `json:"reload_semantics"`
	ProcessRestartRequired bool                              `json:"process_restart_required"`
	HotReloadClaim         bool                              `json:"hot_reload_claim"`
	Watch                  bool                              `json:"watch"`
	SupportedTargets       []string                          `json:"supported_targets"`
	Steps                  []surfaceDevStep                  `json:"steps"`
	SourceDiagnostics      []surfaceDevDiagnostic            `json:"source_diagnostics"`
	MorphToPixels          *surface.MorphToPixelsChainReport `json:"morph_to_pixels,omitempty"`
	NegativeGuards         surfaceDevGuards                  `json:"negative_guards"`
	Pass                   bool                              `json:"pass"`
}

type surfaceDevStep struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	ChangedPath     string   `json:"changed_path"`
	OutputPath      string   `json:"output_path"`
	DurationMS      int64    `json:"duration_ms"`
	CompiledModules []string `json:"compiled_modules"`
	CacheHits       []string `json:"cache_hits"`
	Pass            bool     `json:"pass"`
	Error           string   `json:"error,omitempty"`
}

type surfaceDevDiagnostic struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Pass     bool   `json:"pass"`
}

type surfaceDevGuards struct {
	NoHotReloadClaim                   bool `json:"no_hot_reload_claim"`
	FullRestartDocumentedAsFastRebuild bool `json:"full_restart_documented_as_fast_rebuild"`
	NoElectronDevServer                bool `json:"no_electron_dev_server"`
	NoReactFastRefresh                 bool `json:"no_react_fast_refresh"`
	NoDOMHotReload                     bool `json:"no_dom_hot_reload"`
}

type surfaceDevChange struct {
	kind string
	path string
}

func runSurface(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printSurfaceUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printSurfaceUsage(stdout)
		return 0
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "run":
		return runSurfaceRun(args[1:], stdout, stderr)
	case "dev":
		return runSurfaceDev(args[1:], stdout, stderr)
	case "inspect":
		return runSurfaceInspectCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown surface command %q\n", args[0])
		printSurfaceUsage(stderr)
		return 2
	}
}

func printSurfaceUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra surface <check|run|dev|inspect> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  check  run Surface-related semantic checks")
	fmt.Fprintln(w, "  run    run a native Surface app through the Wayland host")
	fmt.Fprintln(w, "  dev    run the scoped Surface fast rebuild developer loop")
	fmt.Fprintln(w, "  inspect  build a Surface inspector snapshot from a runtime report")
}

func runSurfaceInspectCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	reportPath := fs.String("report", "", "Surface runtime report JSON")
	outPath := fs.String("out", "", "write inspector snapshot JSON to this path; defaults to stdout")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface inspect does not accept positional arguments")
		return 2
	}
	if *reportPath == "" {
		fmt.Fprintln(stderr, "--report is required")
		return 2
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	snapshot, err := surfaceinspector.SnapshotFromReportRaw(raw, *reportPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	out, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	out = append(out, '\n')
	if *outPath == "" {
		if _, err := stdout.Write(out); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*outPath, out, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote surface inspector snapshot to %s\n", *outPath)
	return 0
}

func runSurfaceRun(args []string, stdout io.Writer, stderr io.Writer) int {
	runArgs := []string{"--target", "linux-x64", "--surface-host", "wayland"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--host-report":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "surface run --host-report requires a value")
				return 2
			}
			runArgs = append(runArgs, "--surface-host-report", args[i+1])
			i++
		case strings.HasPrefix(arg, "--host-report="):
			runArgs = append(
				runArgs,
				"--surface-host-report",
				strings.TrimPrefix(arg, "--host-report="),
			)
		default:
			runArgs = append(runArgs, arg)
		}
	}
	return runRun(runArgs, stdout, stderr)
}

func runSurfaceDev(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface dev", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sourceFlag := fs.String(
		"source",
		"",
		"Surface app source path; defaults to positional input or discovered project entry",
	)
	targetFlag := fs.String(
		"target",
		"",
		"target triple; current fast rebuild evidence is linux-x64",
	)
	outDirFlag := fs.String("out-dir", "", "directory for dev-loop build artifacts")
	reportPath := fs.String("report", "", "write tetra.surface.dev-workflow.v1 JSON report")
	morphRenderedBeautyReportPath := fs.String(
		"morph-rendered-beauty-report",
		"",
		"attach validated tetra.surface.morph-rendered-beauty.v1 evidence to the dev report",
	)
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	watch := fs.Bool(
		"watch",
		false,
		"reserve watch-mode metadata; current command records one fast rebuild loop",
	)
	var changeFlags multiFlag
	fs.Var(
		&changeFlags,
		"change-file",
		"changed Surface path as kind:path; repeat for token, recipe, and source",
	)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"surface dev accepts at most one input path",
		)
		return 2
	}
	source := strings.TrimSpace(*sourceFlag)
	if source != "" && fs.NArg() == 1 {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"surface dev accepts either --source or one positional input path, not both",
		)
		return 2
	}
	if source == "" && fs.NArg() == 1 {
		source = fs.Arg(0)
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(source)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	rawTarget := strings.TrimSpace(*targetFlag)
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if tgt.Triple != "linux-x64" {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			("surface dev fast rebuild evidence is currently scoped to linux-" +
				"x64; wasm32-web/headless are documented targets without hot reload " +
				"promotion"),
		)
		return 2
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, nil)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions("exe", "auto", false, "", targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots

	outDir, cleanup, err := surfaceDevOutputDir(*outDirFlag)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer cleanup()

	report := newSurfaceDevReport(input, tgt, *watch)
	if strings.TrimSpace(*morphRenderedBeautyReportPath) != "" {
		chain, err := loadSurfaceDevMorphToPixelsChain(*morphRenderedBeautyReportPath, input)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		report.MorphToPixels = chain
	}
	changes, err := parseSurfaceDevChanges(changeFlags)
	if err != nil {
		writeValidationDiagnostic(stderr, *diagnostics, err.Error())
		return 2
	}
	report.SourceDiagnostics = append(
		report.SourceDiagnostics,
		surfaceDevInfoDiagnostic("source", input),
	)
	for _, change := range changes {
		report.SourceDiagnostics = append(
			report.SourceDiagnostics,
			surfaceDevInfoDiagnostic(change.kind, change.path),
		)
	}

	initial := runSurfaceDevBuild("initial build", "initial", "", input, outDir, tgt, opt)
	report.Steps = append(report.Steps, initial)
	if !initial.Pass {
		report.Pass = false
		report.SourceDiagnostics = []surfaceDevDiagnostic{
			surfaceDevErrorDiagnostic(input, initial.Error),
		}
		writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
		writeDiagnostic(stderr, *diagnostics, errors.New(initial.Error))
		return 1
	}
	warm := runSurfaceDevBuild("warm rebuild", "warm-cache", "", input, outDir, tgt, opt)
	report.Steps = append(report.Steps, warm)
	if !warm.Pass {
		report.Pass = false
		report.SourceDiagnostics = []surfaceDevDiagnostic{
			surfaceDevErrorDiagnostic(input, warm.Error),
		}
		writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
		writeDiagnostic(stderr, *diagnostics, errors.New(warm.Error))
		return 1
	}

	restoreFns := make([]func(), 0, len(changes))
	for _, change := range changes {
		restore, err := appendSurfaceDevChange(change)
		if err != nil {
			report.Pass = false
			report.SourceDiagnostics = []surfaceDevDiagnostic{
				surfaceDevErrorDiagnostic(change.path, err.Error()),
			}
			writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		restoreFns = append(restoreFns, restore)
		step := runSurfaceDevBuild(
			surfaceDevStepName(change.kind),
			change.kind+"-change",
			change.path,
			input,
			outDir,
			tgt,
			opt,
		)
		report.Steps = append(report.Steps, step)
		if !step.Pass {
			report.Pass = false
			report.SourceDiagnostics = []surfaceDevDiagnostic{
				surfaceDevErrorDiagnostic(change.path, step.Error),
			}
			writeSurfaceDevReportIfRequested(*reportPath, report, stderr)
			writeDiagnostic(stderr, *diagnostics, errors.New(step.Error))
			for i := len(restoreFns) - 1; i >= 0; i-- {
				restoreFns[i]()
			}
			return 1
		}
	}
	for i := len(restoreFns) - 1; i >= 0; i-- {
		restoreFns[i]()
	}

	report.Pass = surfaceDevReportPass(report)
	if err := writeSurfaceDevReportIfRequested(*reportPath, report, stderr); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if report.Pass {
		fmt.Fprintf(
			stdout,
			"Surface dev fast rebuild report: %s\n",
			defaultReportLabel(*reportPath),
		)
		return 0
	}
	writeDiagnostic(
		stderr,
		*diagnostics,
		errors.New(
			"surface dev fast rebuild report did not satisfy required token/recipe/source evidence",
		),
	)
	return 1
}

func newSurfaceDevReport(input string, tgt ctarget.Target, watch bool) surfaceDevReport {
	return surfaceDevReport{
		Schema:                 "tetra.surface.dev-workflow.v1",
		Model:                  "surface-dev-workflow-v1",
		ReleaseScope:           "surface-v1-linux-web",
		Command:                "tetra surface dev",
		Source:                 filepath.ToSlash(input),
		Target:                 tgt.Triple,
		Mode:                   "fast-rebuild",
		ReloadSemantics:        "fast-rebuild",
		ProcessRestartRequired: true,
		HotReloadClaim:         false,
		Watch:                  watch,
		SupportedTargets:       []string{"headless", "linux-x64", "wasm32-web"},
		NegativeGuards: surfaceDevGuards{
			NoHotReloadClaim:                   true,
			FullRestartDocumentedAsFastRebuild: true,
			NoElectronDevServer:                true,
			NoReactFastRefresh:                 true,
			NoDOMHotReload:                     true,
		},
	}
}

func surfaceDevOutputDir(raw string) (string, func(), error) {
	if strings.TrimSpace(raw) != "" {
		if err := os.MkdirAll(raw, 0o755); err != nil {
			return "", func() {}, err
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			return "", func() {}, err
		}
		return abs, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "tetra-surface-dev-*")
	if err != nil {
		return "", func() {}, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

func parseSurfaceDevChanges(values []string) ([]surfaceDevChange, error) {
	changes := make([]surfaceDevChange, 0, len(values))
	seen := map[string]bool{}
	for _, raw := range values {
		kind, path, ok := strings.Cut(raw, ":")
		if !ok {
			return nil, fmt.Errorf("surface dev --change-file must use kind:path")
		}
		kind = strings.TrimSpace(kind)
		path = strings.TrimSpace(path)
		switch kind {
		case "token", "recipe", "source", "block", "morph":
		default:
			return nil, fmt.Errorf("surface dev --change-file kind %q is unsupported", kind)
		}
		if path == "" {
			return nil, fmt.Errorf("surface dev --change-file path is required")
		}
		if seen[kind] {
			return nil, fmt.Errorf("surface dev --change-file duplicate kind %q", kind)
		}
		seen[kind] = true
		changes = append(changes, surfaceDevChange{kind: kind, path: path})
	}
	sort.Slice(changes, func(i, j int) bool {
		order := map[string]int{"token": 0, "recipe": 1, "source": 2, "block": 3, "morph": 4}
		return order[changes[i].kind] < order[changes[j].kind]
	})
	return changes, nil
}

func runSurfaceDevBuild(
	name string,
	kind string,
	changedPath string,
	input string,
	outDir string,
	tgt ctarget.Target,
	opt compiler.BuildOptions,
) surfaceDevStep {
	outputPath := filepath.Join(outDir, kind, "app"+tgt.ExeExt)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return surfaceDevStep{
			Name:        name,
			Kind:        kind,
			ChangedPath: filepath.ToSlash(changedPath),
			OutputPath:  filepath.ToSlash(outputPath),
			Pass:        false,
			Error:       err.Error(),
		}
	}
	start := time.Now()
	stats, err := compiler.BuildFileWithStatsOpt(input, outputPath, tgt.Triple, opt)
	duration := time.Since(start).Milliseconds()
	step := surfaceDevStep{
		Name:        name,
		Kind:        kind,
		ChangedPath: filepath.ToSlash(changedPath),
		OutputPath:  filepath.ToSlash(outputPath),
		DurationMS:  duration,
		Pass:        err == nil,
	}
	if stats != nil {
		step.CompiledModules = append([]string(nil), stats.CompiledModules...)
		step.CacheHits = append([]string(nil), stats.CacheHits...)
		sort.Strings(step.CompiledModules)
		sort.Strings(step.CacheHits)
	}
	if err != nil {
		step.Error = err.Error()
	}
	return step
}

func appendSurfaceDevChange(change surfaceDevChange) (func(), error) {
	raw, err := os.ReadFile(change.path)
	if err != nil {
		return func() {}, err
	}
	next := append([]byte(nil), raw...)
	next = append(
		next,
		[]byte(
			fmt.Sprintf(
				"\n// tetra surface dev %s fast-rebuild change %d\n",
				change.kind,
				time.Now().UnixNano(),
			),
		)...)
	if err := os.WriteFile(change.path, next, 0o644); err != nil {
		return func() {}, err
	}
	return func() { _ = os.WriteFile(change.path, raw, 0o644) }, nil
}

func surfaceDevInfoDiagnostic(kind string, path string) surfaceDevDiagnostic {
	if kind == "" {
		kind = classifySurfaceDevPath(path, nil)
	}
	codeKind := strings.ToUpper(strings.ReplaceAll(kind, "-", "_"))
	return surfaceDevDiagnostic{
		Kind:     kind,
		Path:     filepath.ToSlash(path),
		Line:     1,
		Column:   1,
		Code:     "SURFACE_DEV_" + codeKind + "_PATH",
		Message:  kind + " file participates in Surface fast rebuild",
		Severity: "info",
		Pass:     true,
	}
}

func surfaceDevErrorDiagnostic(path string, message string) surfaceDevDiagnostic {
	diag := compiler.DiagnosticFromError(errors.New(message))
	kind := classifySurfaceDevPath(path, nil)
	line := diag.Line
	column := diag.Column
	if line <= 0 {
		line = 1
	}
	if column <= 0 {
		column = 1
	}
	diagPath := diag.File
	if diagPath == "" {
		diagPath = path
	}
	return surfaceDevDiagnostic{
		Kind:     kind,
		Path:     filepath.ToSlash(diagPath),
		Line:     line,
		Column:   column,
		Code:     diag.Code,
		Message:  diag.Message,
		Severity: "error",
		Pass:     false,
	}
}

func classifySurfaceDevPath(path string, content []byte) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	if len(content) == 0 {
		if raw, err := os.ReadFile(path); err == nil {
			content = raw
		}
	}
	text := strings.ToLower(string(content))
	switch {
	case strings.Contains(lower, "token") || strings.Contains(text, "token"):
		return "token"
	case strings.Contains(lower, "recipe") || strings.Contains(text, "recipe"):
		return "recipe"
	case strings.Contains(lower, "morph") || strings.Contains(text, "morph"):
		return "morph"
	case strings.Contains(lower, "block") || strings.Contains(text, "block"):
		return "block"
	default:
		return "source"
	}
}

func surfaceDevStepName(kind string) string {
	switch kind {
	case "token":
		return "token rebuild"
	case "recipe":
		return "recipe rebuild"
	case "source":
		return "source rebuild"
	case "block":
		return "block rebuild"
	case "morph":
		return "morph rebuild"
	default:
		return kind + " rebuild"
	}
}

func surfaceDevReportPass(report surfaceDevReport) bool {
	kinds := map[string]surfaceDevStep{}
	for _, step := range report.Steps {
		if !step.Pass {
			return false
		}
		kinds[step.Kind] = step
	}
	warm, ok := kinds["warm-cache"]
	if !ok || len(warm.CompiledModules) != 0 || len(warm.CacheHits) == 0 {
		return false
	}
	for _, kind := range []string{"initial", "token-change", "recipe-change", "source-change"} {
		step, ok := kinds[kind]
		if !ok {
			return false
		}
		if kind != "initial" && len(step.CompiledModules) == 0 {
			return false
		}
	}
	diagKinds := map[string]bool{}
	for _, diag := range report.SourceDiagnostics {
		diagKinds[diag.Kind] = diag.Pass
	}
	if report.MorphToPixels != nil && !report.MorphToPixels.Pass {
		return false
	}
	return diagKinds["token"] && diagKinds["recipe"] && diagKinds["source"]
}

func loadSurfaceDevMorphToPixelsChain(
	path string,
	expectedSource string,
) (*surface.MorphToPixelsChainReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Morph rendered beauty report %s: %w", path, err)
	}
	if err := surface.ValidateMorphRenderedBeautyReport(raw); err != nil {
		return nil, fmt.Errorf("validate Morph rendered beauty report %s: %w", path, err)
	}
	var report surface.MorphRenderedBeautyReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, fmt.Errorf("decode Morph rendered beauty report %s: %w", path, err)
	}
	chain := surface.MorphToPixelsChainFromRenderedBeauty(filepath.ToSlash(path), report)
	if err := surface.ValidateMorphToPixelsChainReport(chain, expectedSource); err != nil {
		return nil, fmt.Errorf("validate Morph-to-pixels dev evidence %s: %w", path, err)
	}
	return &chain, nil
}

func writeSurfaceDevReportIfRequested(
	path string,
	report surfaceDevReport,
	stderr io.Writer,
) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := writeJSON(path, report); err != nil {
		fmt.Fprintf(stderr, "surface dev report write failed: %v\n", err)
		return err
	}
	return nil
}

func defaultReportLabel(path string) string {
	if strings.TrimSpace(path) == "" {
		return "(not written)"
	}
	return path
}

// ---- test_command.go ----

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	reportFormat := fs.String("report", "text", "report format: text, json, or toon")
	format := fs.String("format", "", "output format alias for --report: text, json, or toon")
	allTargets := fs.Bool("all-targets", false, "run the required x86/x64/x32 target matrix")
	brutal := fs.Bool("brutal", false, "run the full brutal target matrix")
	abiSuite := fs.Bool("abi", false, "run ABI torture tests for the target")
	atomicStress := fs.Bool("atomic-stress", false, "run atomic stress tests for the target")
	fuzzSuite := fs.Bool("fuzz", false, "run fuzz/property tests for the target")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	reportFormatValue, ok := resolveTestReportFormat(
		fs,
		*reportFormat,
		*format,
		*diagnostics,
		stderr,
	)
	if !ok {
		return 2
	}
	if reportFormatValue != outputformat.Text && !outputformat.Structured(reportFormatValue) {
		writeValidationDiagnostic(stderr, *diagnostics, "unsupported --report format")
		return 2
	}
	if *allTargets || *brutal {
		return runAllTargetsSuite(
			*allTargets,
			*brutal,
			*diagnostics,
			reportFormatValue,
			stdout,
			stderr,
		)
	}
	tgt, ok := parseBuildTargetOrReport(*target, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if *abiSuite && !*atomicStress && !*fuzzSuite {
		return runTargetABISuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *atomicStress && !*abiSuite && !*fuzzSuite {
		return runTargetAtomicStressSuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *fuzzSuite && !*abiSuite && !*atomicStress {
		return runTargetFuzzSuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *abiSuite || *atomicStress || *fuzzSuite {
		writeDiagnostic(
			stderr,
			*diagnostics,
			unsupportedTargetTestSuiteDiagnostic(tgt.Triple, *abiSuite, *atomicStress, *fuzzSuite),
		)
		return 2
	}
	paths := fs.Args()
	explicitPaths := len(paths) > 0
	explicitTarget := testArgsIncludeTargetFlag(args)
	explicitSingleFileInput := false
	if len(paths) == 1 {
		if info, err := os.Stat(paths[0]); err == nil && !info.IsDir() {
			explicitSingleFileInput = true
		}
	}
	var projectCtx *cliProjectContext
	var worldOpt compiler.WorldOptions
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			projectCtx = ctx
			worldOpt = compiler.WorldOptions{
				Root:            ctx.Root,
				SourceRoots:     append([]string(nil), ctx.SourceRoots...),
				DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
			}
			paths = existingProjectSourcePaths(ctx)
		}
		if len(paths) == 0 {
			if explicitTarget && isRequiredTargetSuiteTriple(tgt.Triple) {
				return runTargetDefaultSuite(tgt, reportFormatValue, stdout, stderr)
			}
			paths = []string{"."}
		}
	} else if len(paths) == 1 {
		resolved, resolvedWorldOpt, ctx, err := resolveCLIInput(paths[0])
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found && isProjectReference(paths[0], ctx) {
			projectCtx = ctx
			worldOpt = resolvedWorldOpt
			paths = existingProjectSourcePaths(ctx)
			if len(paths) == 0 {
				paths = []string{resolved}
			}
		}
	}
	if isWASMTargetTriple(tgt.Triple) {
		writeTargetRuntimeDiagnostic(
			stderr,
			*diagnostics,
			fmt.Sprintf(
				("cannot run tests for target %s: WASM test runner is not part of "+
					"the current production runtime contract; use smoke/runtime reports for "+
					"WASM execution evidence"),
				tgt.Triple,
			),
		)
		return 2
	}
	if ctarget.IsBuildOnlyTarget(tgt.Triple) && !canRunBuildOnlyNativeTargetOnHost(tgt) {
		reason := buildOnlyNativeRunUnsupportedReason(tgt)
		writeTargetRuntimeDiagnostic(
			stderr,
			*diagnostics,
			fmt.Sprintf("cannot run tests for target %s: %s", tgt.Triple, reason),
		)
		return 2
	}
	if !canRunNativeExecutableTargetOnHost(tgt) {
		writeTargetRuntimeDiagnostic(
			stderr,
			*diagnostics,
			fmt.Sprintf(
				"cannot run tests for target %s on host %s/%s",
				tgt.Triple,
				runtime.GOOS,
				runtime.GOARCH,
			),
		)
		return 2
	}
	if !explicitPaths && explicitTarget && projectCtx == nil &&
		isRequiredTargetSuiteTriple(tgt.Triple) {
		return runTargetDefaultSuite(tgt, reportFormatValue, stdout, stderr)
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, nil)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-test-*")
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	total := 0
	passed := 0
	var results []compiler.TestRunnerResult
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		runners, err := compiler.TestRunnerSources(raw, file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		for i, runner := range runners {
			total++
			start := time.Now()
			srcPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.t4", total))
			runnerSource := runner.Source
			sourceModule := modulePathFromSource(runner.Source)
			if sourceModule != "" {
				var err error
				srcPath, runnerSource, err = runnerSourcePathForModuleFile(
					file,
					runner.Source,
					total,
				)
				if err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
				defer os.Remove(srcPath)
			}
			outPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d%s", total, tgt.ExeExt))
			if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if err := os.WriteFile(srcPath, runnerSource, 0o644); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if modulePathFromSource(runnerSource) != "" && projectCtx != nil && projectCtx.Found {
				runnerProjectCtx := projectCtx
				runnerWorldOpt := worldOpt
				runnerLinkObjects := targetLinkObjects
				if runnerProjectCtx == nil {
					var err error
					runnerProjectCtx, runnerWorldOpt, runnerLinkObjects, err = testProjectContextForFile(
						file,
						sourceModule,
						tgt.Triple,
					)
					if err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				}
				if runnerProjectCtx != nil && runnerProjectCtx.Found {
					opt := compiler.BuildOptions{
						Jobs:        1,
						ProjectRoot: runnerWorldOpt.Root,
						SourceRoots: append([]string(nil), runnerWorldOpt.SourceRoots...),
						DependencyRoots: append(
							[]compiler.ModuleRoot(nil),
							runnerWorldOpt.DependencyRoots...),
						LinkObjectPaths: append([]string(nil), runnerLinkObjects...),
					}
					if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				} else if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else if modulePathFromSource(runnerSource) != "" {
				var runnerProjectCtx *cliProjectContext
				var runnerWorldOpt compiler.WorldOptions
				var runnerLinkObjects []string
				if !explicitSingleFileInput {
					var err error
					runnerProjectCtx, runnerWorldOpt, runnerLinkObjects, err = testProjectContextForFile(
						file,
						sourceModule,
						tgt.Triple,
					)
					if err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				}
				if runnerProjectCtx != nil && runnerProjectCtx.Found {
					opt := compiler.BuildOptions{
						Jobs:            1,
						ProjectRoot:     runnerWorldOpt.Root,
						SourceRoots:     append([]string(nil), runnerWorldOpt.SourceRoots...),
						DependencyRoots: append([]compiler.ModuleRoot(nil), runnerWorldOpt.DependencyRoots...),
						LinkObjectPaths: append([]string(nil), runnerLinkObjects...),
					}
					if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				} else if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else {
				if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			code := execNativeProgram(outPath, io.Discard, io.Discard)
			name := runner.Name
			if name == "" {
				name = fmt.Sprintf("%s#%d", file, i+1)
			}
			result := runner.ResultWithDuration(code, nil, elapsedMillis(time.Since(start)))
			results = append(results, result)
			if code == 0 {
				passed++
				if reportFormatValue == "text" {
					fmt.Fprintf(stdout, "PASS %s\n", name)
				}
			} else {
				if reportFormatValue == "text" {
					if result.Error != "" {
						fmt.Fprintf(stdout, "FAIL %s (%s)\n", name, result.Error)
					} else {
						fmt.Fprintf(stdout, "FAIL %s\n", name)
					}
				}
			}
		}
	}
	if outputformat.Structured(reportFormatValue) {
		if err := outputformat.WriteStructured(
			stdout,
			reportFormatValue,
			compiler.NewTestRunnerReportForTarget(results, tgt.Triple),
		); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, total)
	}
	if passed != total {
		return 1
	}
	return 0
}

func resolveTestReportFormat(
	fs *flag.FlagSet,
	reportValue string,
	formatValue string,
	diagnostics string,
	stderr io.Writer,
) (string, bool) {
	reportProvided := false
	formatProvided := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "report":
			reportProvided = true
		case "format":
			formatProvided = true
		}
	})
	if !formatProvided {
		return reportValue, true
	}
	if formatValue != outputformat.Text && !outputformat.Structured(formatValue) {
		writeValidationDiagnostic(stderr, diagnostics, "unsupported --format")
		return "", false
	}
	if reportProvided && reportValue != formatValue {
		writeValidationDiagnostic(
			stderr,
			diagnostics,
			"--format and --report must match when both are provided",
		)
		return "", false
	}
	return formatValue, true
}

func testProjectContextForFile(
	file string,
	module string,
	target string,
) (*cliProjectContext, compiler.WorldOptions, []string, error) {
	ctx, err := discoverCLIProject(filepath.Dir(file))
	if err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	if ctx == nil || !ctx.Found || !fileModuleMatchesProjectSourceRoots(file, module, ctx) {
		return nil, compiler.WorldOptions{}, nil, nil
	}
	if err := validateDiscoveredProjectLock(ctx, target); err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	linkObjects, err := projectLinkObjects(ctx, target, nil)
	if err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	opt := compiler.WorldOptions{
		Root:            ctx.Root,
		SourceRoots:     append([]string(nil), ctx.SourceRoots...),
		DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
	}
	return ctx, opt, linkObjects, nil
}

func fileModuleMatchesProjectSourceRoots(file string, module string, ctx *cliProjectContext) bool {
	if module == "" || ctx == nil || !ctx.Found {
		return false
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(ctx.Root, abs)
	if err != nil {
		return false
	}
	cleanRel := filepath.Clean(rel)
	if cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(cleanRel) {
		return false
	}
	for _, root := range ctx.SourceRoots {
		cleanRoot := filepath.Clean(filepath.FromSlash(root))
		moduleRel := cleanRel
		if root != "" && cleanRoot != "." {
			if cleanRel == cleanRoot ||
				!strings.HasPrefix(cleanRel, cleanRoot+string(filepath.Separator)) {
				continue
			}
			moduleRel = strings.TrimPrefix(cleanRel, cleanRoot+string(filepath.Separator))
		}
		if cliModuleRelPathMatches(module, moduleRel) {
			return true
		}
	}
	return false
}

func testArgsIncludeTargetFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--target" || arg == "-target" || strings.HasPrefix(arg, "--target=") ||
			strings.HasPrefix(arg, "-target=") {
			return true
		}
	}
	return false
}

func isRequiredTargetSuiteTriple(triple string) bool {
	switch triple {
	case "linux-x86", "linux-x64", "linux-x32":
		return true
	default:
		return false
	}
}

func runTargetDefaultSuite(
	tgt ctarget.Target,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	return writeTargetSuiteReport(runABISuiteResults(tgt), tgt.Triple, reportFormat, stdout, stderr)
}

func unsupportedTargetTestSuiteDiagnostic(
	target string,
	abiSuite bool,
	atomicStress bool,
	fuzzSuite bool,
) error {
	var suites []string
	var flags []string
	if abiSuite {
		suites = append(suites, "ABI torture")
		flags = append(flags, "--abi")
	}
	if atomicStress {
		suites = append(suites, "atomic stress")
		flags = append(flags, "--atomic-stress")
	}
	if fuzzSuite {
		suites = append(suites, "fuzz")
		flags = append(flags, "--fuzz")
	}
	return fmt.Errorf(
		("test suite %s (%s) for target %s is not implemented yet: no " +
			"real target runner or oracle is wired; no fake or skipped tests will be " +
			"emitted"),
		strings.Join(suites, ", "),
		strings.Join(flags, ", "),
		target,
	)
}

func runAllTargetsSuite(
	allTargets bool,
	brutal bool,
	diagnostics string,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	targets := []string{"x86", "x64", "macos-x64", "windows-x64", "x32"}
	results := make([]compiler.TestRunnerResult, 0, 55)
	for _, raw := range targets {
		tgt, err := ctarget.Parse(raw)
		if err != nil {
			results = append(
				results,
				targetSuiteResult(0, "tetra:all-targets", raw+" target parse", 1, err),
			)
			continue
		}
		results = append(results, runABISuiteResults(tgt)...)
	}
	if brutal {
		for _, raw := range targets {
			tgt, err := ctarget.Parse(raw)
			if err != nil {
				results = append(
					results,
					targetSuiteResult(0, "tetra:atomic-stress", raw+" atomic target parse", 1, err),
				)
				continue
			}
			results = append(results, runAtomicStressSuiteResults(tgt)...)
		}
		for _, raw := range targets {
			tgt, err := ctarget.Parse(raw)
			if err != nil {
				results = append(
					results,
					targetSuiteResult(0, "tetra:fuzz", raw+" fuzz target parse", 1, err),
				)
				continue
			}
			results = append(results, runFuzzSuiteResults(tgt)...)
		}
	}
	return writeTargetSuiteReport(results, "", reportFormat, stdout, stderr)
}

func runABISuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	if tgt.Triple == "linux-x86" {
		results, err := runX86ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{
				targetSuiteResult(0, "tetra:x86-abi", "x86 ABI suite", 1, err),
			}
		}
		return results
	}
	if tgt.Triple == "linux-x64" {
		results, err := runX64ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{
				targetSuiteResult(0, "tetra:x64-abi", "x64 ABI suite", 1, err),
			}
		}
		return results
	}
	if tgt.Triple == "linux-x32" {
		results, err := runX32ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{
				targetSuiteResult(0, "tetra:x32-abi", "x32 ABI suite", 1, err),
			}
		}
		return results
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{
			targetSuiteResult(
				0,
				fmt.Sprintf("tetra:%s-abi", tgt.Arch),
				tgt.Arch.String()+" ABI suite",
				1,
				err,
			),
		}
	}
	return targetABICheckResults(tgt, checks)
}

func runAtomicStressSuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	checks, err := compiler.RunTargetAtomicStressChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{
			targetSuiteResult(
				0,
				fmt.Sprintf("tetra:%s-atomic-stress", tgt.Arch),
				tgt.Arch.String()+" atomic stress",
				1,
				err,
			),
		}
	}
	return targetAtomicStressCheckResults(tgt, checks)
}

func runFuzzSuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	checks, err := compiler.RunTargetFuzzChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{
			targetSuiteResult(
				0,
				fmt.Sprintf("tetra:%s-fuzz", tgt.Arch),
				tgt.Arch.String()+" fuzz",
				1,
				err,
			),
		}
	}
	return targetFuzzCheckResults(tgt, checks)
}

func unsupportedMatrixResult(
	filename string,
	name string,
	message string,
) compiler.TestRunnerResult {
	return targetSuiteResult(0, filename, name, 1, fmt.Errorf("%s", message))
}

func runTargetABISuite(
	targetName string,
	diagnostics string,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	if tgt.Triple == "linux-x86" {
		results, err := runX86ABISuite()
		if err != nil {
			writeDiagnostic(stderr, diagnostics, err)
			return 1
		}
		return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
	}
	if tgt.Triple == "linux-x64" {
		results, err := runX64ABISuite()
		if err != nil {
			writeDiagnostic(stderr, diagnostics, err)
			return 1
		}
		return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
	}
	if tgt.Triple != "linux-x32" {
		checks, err := compiler.RunTargetABIChecks(tgt.Triple)
		if err != nil {
			writeDiagnostic(
				stderr,
				diagnostics,
				unsupportedTargetTestSuiteDiagnostic(tgt.Triple, true, false, false),
			)
			return 2
		}
		return writeTargetSuiteReport(
			targetABICheckResults(tgt, checks),
			tgt.Triple,
			reportFormat,
			stdout,
			stderr,
		)
	}
	results, err := runX32ABISuite()
	if err != nil {
		writeDiagnostic(stderr, diagnostics, err)
		return 1
	}
	return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
}

func runTargetAtomicStressSuite(
	targetName string,
	diagnostics string,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	checks, err := compiler.RunTargetAtomicStressChecks(tgt.Triple)
	if err != nil {
		writeDiagnostic(
			stderr,
			diagnostics,
			unsupportedTargetTestSuiteDiagnostic(tgt.Triple, false, true, false),
		)
		return 2
	}
	return writeTargetSuiteReport(
		targetAtomicStressCheckResults(tgt, checks),
		tgt.Triple,
		reportFormat,
		stdout,
		stderr,
	)
}

func runTargetFuzzSuite(
	targetName string,
	diagnostics string,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	checks, err := compiler.RunTargetFuzzChecks(tgt.Triple)
	if err != nil {
		writeDiagnostic(
			stderr,
			diagnostics,
			unsupportedTargetTestSuiteDiagnostic(tgt.Triple, false, false, true),
		)
		return 2
	}
	return writeTargetSuiteReport(
		targetFuzzCheckResults(tgt, checks),
		tgt.Triple,
		reportFormat,
		stdout,
		stderr,
	)
}

func writeTargetSuiteReport(
	results []compiler.TestRunnerResult,
	target string,
	reportFormat string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	passed := 0
	for _, result := range results {
		if result.Passed {
			passed++
			if reportFormat == "text" {
				fmt.Fprintf(stdout, "PASS %s\n", result.Name)
			}
			continue
		}
		if reportFormat == "text" {
			if result.Error != "" {
				fmt.Fprintf(stdout, "FAIL %s (%s)\n", result.Name, result.Error)
			} else {
				fmt.Fprintf(stdout, "FAIL %s\n", result.Name)
			}
		}
	}
	if outputformat.Structured(reportFormat) {
		if err := outputformat.WriteStructured(
			stdout,
			reportFormat,
			compiler.NewTestRunnerReportForTarget(results, target),
		); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, len(results))
	}
	if passed != len(results) {
		return 1
	}
	return 0
}

func runX32ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x32-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x32")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x32 object ABI smoke", run: runX32ObjectABISmoke},
		{name: "x32 atomic ABI object", run: runX32AtomicABIObject},
		{name: "x32 executable matrix smoke", run: runX32ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(
			results,
			targetSuiteResult(
				len(checks)+i,
				"tetra:x32-abi",
				tc.name,
				elapsedMillis(time.Since(start)),
				err,
			),
		)
	}
	return results, nil
}

func targetABICheckResults(
	tgt ctarget.Target,
	checks []compiler.ABICheck,
) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-abi", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetAtomicStressCheckResults(
	tgt ctarget.Target,
	checks []compiler.AtomicStressCheck,
) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-atomic-stress", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetFuzzCheckResults(
	tgt ctarget.Target,
	checks []compiler.FuzzCheck,
) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-fuzz", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetSuiteFilenameStem(tgt ctarget.Target) string {
	switch tgt.Triple {
	case "linux-x86":
		return "x86"
	case "linux-x64":
		return "x64"
	case "linux-x32":
		return "x32"
	default:
		return tgt.Triple
	}
}

func targetSuiteResult(
	index int,
	filename string,
	name string,
	durationMS int64,
	err error,
) compiler.TestRunnerResult {
	result := compiler.TestRunnerResult{
		Name:         name,
		Filename:     filename,
		Index:        index,
		FunctionName: targetSuiteFunctionName(name),
		Passed:       err == nil,
		DurationMS:   durationMS,
	}
	if err != nil {
		result.ExitCode = 1
		result.Error = err.Error()
	}
	return result
}

func targetSuiteFunctionName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_", ":", "_")
	return "__tetra_test_" + replacer.Replace(strings.ToLower(strings.TrimSpace(name)))
}

func runX86ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x86")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x86 object ABI smoke", run: runX86ObjectABISmoke},
		{
			name: "x86 atomic ABI object",
			run:  func(string) error { return runAtomicABIObjectCheck("x86", "x86") },
		},
		{name: "x86 executable matrix smoke", run: runX86ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(
			results,
			targetSuiteResult(
				len(checks)+i,
				"tetra:x86-abi",
				tc.name,
				elapsedMillis(time.Since(start)),
				err,
			),
		)
	}
	return results, nil
}

func runX64ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x64")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x64 object ABI smoke", run: runX64ObjectABISmoke},
		{
			name: "x64 atomic ABI object",
			run:  func(string) error { return runAtomicABIObjectCheck("x64", "x64") },
		},
		{name: "x64 executable matrix smoke", run: runX64ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(
			results,
			targetSuiteResult(
				len(checks)+i,
				"tetra:x64-abi",
				tc.name,
				elapsedMillis(time.Since(start)),
				err,
			),
		)
	}
	return results, nil
}

func runAtomicABIObjectCheck(targetName string, prefix string) error {
	checks, err := compiler.RunTargetAtomicStressChecks(targetName)
	if err != nil {
		return err
	}
	want := prefix + " atomic object matrix"
	for _, check := range checks {
		if check.Name != want {
			continue
		}
		if check.Error != "" {
			return fmt.Errorf("%s: %s", want, check.Error)
		}
		return nil
	}
	return fmt.Errorf("atomic object matrix check %q was not produced for %s", want, targetName)
}

func runX64ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x64_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x64_abi_smoke.tobj")
	src := tetraSource(
		`@export("ffi_say_i32")`,
		"fun say(): i32 uses io {",
		`  print("x64 abi\n")`,
		"  return 0",
		"}",
	)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x64",
		compiler.BuildOptions{Emit: compiler.EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x64" {
		return fmt.Errorf("target mismatch: got %q want linux-x64", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 1) || !bytes.Contains(obj.Code, []byte{0x0F, 0x05}) {
		return fmt.Errorf("missing linux-x64 write syscall in object code")
	}
	if containsMovEaxImm32(obj.Code, 0x40000001) {
		return fmt.Errorf("linux-x64 object emitted x32 write syscall number")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf(
				"linux-x64 object unexpectedly has Windows IAT reloc: %#v",
				obj.Relocs,
			)
		}
	}
	return nil
}

func runX64ExecutableMatrixSmoke(tmpDir string) error {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "control",
			src: tetraSource(
				"func add(a: Int, b: Int) -> Int:",
				"    return a + b",
				"",
				"func main() -> Int:",
				"    var i: Int = 0",
				"    var acc: Int = 0",
				"    while i < 3:",
				"        acc = acc + i",
				"        i = i + 1",
				"    return add(acc, 39)",
			),
		},
		{
			name: "aggregates",
			src: tetraSource(
				"struct Pair:",
				"    left: Int",
				"    right: Int",
				"",
				"enum Msg:",
				"    case value(Pair)",
				"    case empty",
				"",
				"func pick() -> Msg:",
				"    return Msg.value(Pair(left: 40, right: 2))",
				"",
				"func main() -> Int:",
				"    match pick():",
				"    case Msg.value(pair):",
				"        return pair.left + pair.right",
				"    case Msg.empty:",
				"        return 0",
			),
		},
		{
			name: "memory",
			src: tetraSource(
				"fun main(): i32 uses alloc, mem {",
				"  var bytes: []u8 = make_u8(2)",
				"  var words: []u16 = make_u16(2)",
				"  var flags: []bool = make_bool(1)",
				"  bytes[0] = 40",
				"  bytes[1] = 1",
				"  words[0] = bytes[0] + bytes[1]",
				"  flags[0] = true",
				"  if flags[0] {",
				"    return words[0] + 1",
				"  }",
				"  return 0",
				"}",
			),
		},
	}
	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, "x64_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x64_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			"x64",
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX64Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX64Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 64 {
		return fmt.Errorf("x64 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x64 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 2 {
		return fmt.Errorf("x64 executable class = %d, want ELFCLASS64", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x64 executable machine = %#x, want EM_X86_64", machine)
	}
	if !containsMovEaxImm32(data, 60) || !bytes.Contains(data, []byte{0x0F, 0x05}) {
		return fmt.Errorf("x64 executable missing x64 exit syscall")
	}
	if containsMovEaxImm32(data, 0x4000003c) {
		return fmt.Errorf("x64 executable emitted x32 exit syscall number")
	}
	if bytes.Contains(data, []byte{0xCD, 0x80}) {
		return fmt.Errorf("x64 executable emitted i386 int 0x80 syscall")
	}
	return nil
}

func runX86ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x86_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x86_abi_smoke.tobj")
	src := tetraSource(
		`@export("ffi_say_i32")`,
		"fun say(): i32 uses io {",
		`  print("x86 abi\n")`,
		"  return 0",
		"}",
	)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x86",
		compiler.BuildOptions{Emit: compiler.EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x86" {
		return fmt.Errorf("target mismatch: got %q want linux-x86", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		return fmt.Errorf("missing i386 write syscall in object code")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf(
				"linux-x86 object unexpectedly has Windows IAT reloc: %#v",
				obj.Relocs,
			)
		}
	}
	return nil
}

type executableMatrixCase struct {
	name string
	src  string
}

func tetraSource(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func x86FamilyExecutableMatrixCases() []executableMatrixCase {
	return []executableMatrixCase{
		{
			name: "recursion",
			src: tetraSource(
				"func fact(n: Int) -> Int:",
				"    if n <= 1:",
				"        return 1",
				"    return n * fact(n - 1)",
				"",
				"func main() -> Int:",
				"    return fact(5)",
			),
		},
		{
			name: "globals_strings",
			src: tetraSource(
				`val greeting: String = "hello"`,
				"var answer: Int = 1",
				"",
				"func main() -> Int:",
				`    let local: String = "abc"`,
				"    answer = greeting.len + local.len + 34",
				"    return answer",
			),
		},
		{
			name: "direct_callback",
			src: tetraSource(
				"func add1(x: Int) -> Int:",
				"    return x + 1",
				"",
				"func apply(cb: fn(Int) -> Int, x: Int) -> Int:",
				"    return cb(x)",
				"",
				"func main() -> Int:",
				"    return apply(add1, 41)",
			),
		},
		{
			name: "control",
			src: tetraSource(
				"func add(a: Int, b: Int) -> Int:",
				"    return a + b",
				"",
				"func main() -> Int:",
				"    var i: Int = 0",
				"    var acc: Int = 0",
				"    while i < 3:",
				"        acc = acc + i",
				"        i = i + 1",
				"    return add(acc, 39)",
			),
		},
		{
			name: "aggregates",
			src: tetraSource(
				"struct Pair:",
				"    left: Int",
				"    right: Int",
				"",
				"enum Msg:",
				"    case value(Pair)",
				"    case empty",
				"",
				"func pick() -> Msg:",
				"    return Msg.value(Pair(left: 40, right: 2))",
				"",
				"func main() -> Int:",
				"    match pick():",
				"    case Msg.value(pair):",
				"        return pair.left + pair.right",
				"    case Msg.empty:",
				"        return 0",
			),
		},
		{
			name: "memory",
			src: tetraSource(
				"fun main(): i32 uses alloc, mem {",
				"  var bytes: []u8 = make_u8(2)",
				"  var words: []u16 = make_u16(2)",
				"  var flags: []bool = make_bool(1)",
				"  bytes[0] = 40",
				"  bytes[1] = 1",
				"  words[0] = bytes[0] + bytes[1]",
				"  flags[0] = true",
				"  if flags[0] {",
				"    return words[0] + 1",
				"  }",
				"  return 0",
				"}",
			),
		},
		{
			name: "raw_memory",
			src: tetraSource(
				"func main() -> Int",
				"uses alloc, capability, mem:",
				"    unsafe:",
				"        let mem: cap.mem = core.cap_mem()",
				"        let p: ptr = core.alloc_bytes(16)",
				"        let stored: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)",
				"        let value: Int = core.load_i32(core.ptr_add(p, 4, mem), mem)",
				"        let stored_ptr: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)",
				"        let loaded_ptr: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)",
				"        if value == 42:",
				"            return 0",
				"    return 1",
			),
		},
		{
			name: "scoped_island",
			src: tetraSource(
				"fun main(): i32 uses alloc, islands, mem {",
				"  var out: i32 = 0",
				"  island(64) as isl {",
				"    var xs: []u16 = core.island_make_u16(isl, 2)",
				"    xs[0] = 40",
				"    xs[1] = 2",
				"    out = xs[0] + xs[1]",
				"  }",
				"  return out",
				"}",
			),
		},
		{
			name: "mmio",
			src: tetraSource(
				"func main() -> Int",
				"uses alloc, capability, io, mem, mmio:",
				"    unsafe:",
				"        let mem: cap.mem = core.cap_mem()",
				"        let io_cap: cap.io = core.cap_io()",
				"        let p: ptr = core.alloc_bytes(4)",
				"        let stored: Int = core.store_i32(p, 41, mem)",
				"        let written: Int = core.mmio_write_i32(p, 42, io_cap)",
				"        return core.mmio_read_i32(p, io_cap)",
				"    return 0",
			),
		},
	}
}

func runX86ExecutableMatrixSmoke(tmpDir string) error {
	for _, tc := range x86FamilyExecutableMatrixCases() {
		srcPath := filepath.Join(tmpDir, "x86_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x86_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			"x86",
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX86Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX86Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 52 {
		return fmt.Errorf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 3 {
		return fmt.Errorf("x86 executable machine = %#x, want EM_386", machine)
	}
	if !bytes.Contains(data, []byte{0x89, 0xC3, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		return fmt.Errorf("x86 executable missing i386 exit int 0x80 stub")
	}
	if containsMovEaxImm32(data, 60) {
		return fmt.Errorf("x86 executable emitted x64 exit syscall number")
	}
	return nil
}

func runX32ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x32_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x32_abi_smoke.tobj")
	src := tetraSource(
		`@export("ffi_say_i32")`,
		"fun say(): i32 uses io {",
		`  print("x32 abi\n")`,
		"  return 0",
		"}",
	)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x32",
		compiler.BuildOptions{Emit: compiler.EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 0x40000001) {
		return fmt.Errorf("missing x32 write syscall number in object code")
	}
	if containsMovEaxImm32(obj.Code, 1) {
		return fmt.Errorf("linux-x32 object emitted plain x64 write syscall number")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf(
				"linux-x32 object unexpectedly has Windows IAT reloc: %#v",
				obj.Relocs,
			)
		}
	}
	return nil
}

func runX32AtomicABIObject(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x32_atomic_abi.tetra")
	outPath := filepath.Join(tmpDir, "x32_atomic_abi.tobj")
	src := `
func atomic_probe() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak64, mem)
        return core.atomic_compare_exchange_weak_i32_seq_cst(p, 0, 1, mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x32",
		compiler.BuildOptions{Emit: compiler.EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !abiObjectHasSymbol(obj, "atomic_probe") {
		return fmt.Errorf("object missing atomic_probe symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07}) {
		return fmt.Errorf("missing qword weak-CAS codegen for i64 atomic on x32")
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07}) {
		return fmt.Errorf("missing dword weak-CAS codegen for i32 atomic on x32")
	}
	return nil
}

func runX32ExecutableMatrixSmoke(tmpDir string) error {
	for _, tc := range x86FamilyExecutableMatrixCases() {
		srcPath := filepath.Join(tmpDir, "x32_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x32_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			"x32",
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX32Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX32Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 52 {
		return fmt.Errorf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x32 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x32 executable machine = %#x, want EM_X86_64", machine)
	}
	if !containsMovEaxImm32(data, 0x4000003c) {
		return fmt.Errorf("x32 executable missing x32 exit syscall number")
	}
	if containsMovEaxImm32(data, 60) {
		return fmt.Errorf("x32 executable emitted plain x64 exit syscall number")
	}
	if bytes.Contains(data, []byte{0xCD, 0x80}) {
		return fmt.Errorf("x32 executable emitted i386 int 0x80 syscall")
	}
	return nil
}

func abiObjectHasSymbol(obj *compiler.Object, name string) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if strings.EqualFold(sym.Name, name) || sym.Name == name {
			return true
		}
	}
	return false
}

func abiObjectHasSymbolSignature(obj *compiler.Object, name string, params int, returns int) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if !(strings.EqualFold(sym.Name, name) || sym.Name == name) {
			continue
		}
		return sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns
	}
	return false
}

func containsMovEaxImm32(buf []byte, imm uint32) bool {
	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] == 0xB8 && binary.LittleEndian.Uint32(buf[i+1:i+5]) == imm {
			return true
		}
	}
	return false
}

func elapsedMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}
