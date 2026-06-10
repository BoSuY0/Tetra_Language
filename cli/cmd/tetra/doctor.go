package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

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
	format := fs.String("format", "text", "output format: text or json")
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
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
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
		return doctorReport{Status: "fail", Checks: []doctorCheck{failCheck("project capsule", err.Error())}}
	}
	if ctx == nil || !ctx.Found {
		return doctorReport{Status: "fail", Checks: []doctorCheck{failCheck("project capsule", "not found")}}
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
		checks = append(checks, passCheck("project dependencies", fmt.Sprintf("%d root(s)", len(ctx.DependencyRoots))))
	}
	if ctx.LockPath == "" {
		checks = append(checks, passCheck("project lock", "not present; run "+projectSyncRepairCommand(ctx.Root, "", false)))
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
		pathCheck(root, "examples/flow_hello.tetra"),
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
		return failCheck("docs manifest version", fmt.Sprintf("got %s want %s", manifest.CompilerVersion, compiler.Version()))
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
		return failCheck("docs manifest surface", fmt.Sprintf("targets got %s want %s", strings.Join(sortedDoctorStrings(targetTriples), ", "), strings.Join(sortedDoctorStrings(buildableTargets), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsSupportedTargets, ctarget.ActorRuntimeTriples()) {
		return failCheck("docs manifest surface", fmt.Sprintf("actors targets got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsSupportedTargets), ", "), strings.Join(sortedDoctorStrings(ctarget.ActorRuntimeTriples()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.ActorsRequiredSymbols, actorRuntimeSymbols()) {
		return failCheck("docs manifest surface", fmt.Sprintf("runtime symbols got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.ActorsRequiredSymbols), ", "), strings.Join(sortedDoctorStrings(actorRuntimeSymbols()), ", ")))
	}
	if !sameStringSet(manifest.RuntimeABI.TimeRequiredSymbols, timeRuntimeSymbols()) {
		return failCheck("docs manifest surface", fmt.Sprintf("time runtime symbols got %s want %s", strings.Join(sortedDoctorStrings(manifest.RuntimeABI.TimeRequiredSymbols), ", "), strings.Join(sortedDoctorStrings(timeRuntimeSymbols()), ", ")))
	}
	return passCheck("docs manifest surface", fmt.Sprintf("%d formats, %d targets, %d runtime symbols", len(manifest.Formats), len(targetTriples), len(actorRuntimeSymbols())+len(timeRuntimeSymbols())))
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
	return passCheck("runtime exports", fmt.Sprintf("%d files, %d symbols", len(paths), len(required)))
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
		if entry.OS != tgt.OS.String() || entry.Arch != tgt.Arch.String() || entry.ABI != tgt.ABI.String() || entry.DataModel != tgt.DataModel.String() || entry.Format != tgt.Format.String() {
			return failCheck("target metadata", fmt.Sprintf("%s metadata mismatch", entry.Triple))
		}
		if entry.PointerWidthBits != tgt.PointerWidthBits || entry.RegisterWidthBits != tgt.RegisterWidthBits || entry.NativeIntWidthBits != tgt.NativeIntWidthBits {
			return failCheck("target metadata", fmt.Sprintf("%s width metadata mismatch", entry.Triple))
		}
		if entry.Endian != tgt.Endian.String() || entry.StackAlignmentBytes != tgt.StackAlignmentBytes || entry.MaxAtomicWidthBits != tgt.MaxAtomicWidthBits {
			return failCheck("target metadata", fmt.Sprintf("%s layout metadata mismatch", entry.Triple))
		}
		if !sameDoctorInts(entry.AtomicWidthBits, tgt.AtomicWidthBits()) || entry.AtomicPointerWidthBits != atomicPointerWidthBits(tgt) {
			return failCheck("target metadata", fmt.Sprintf("%s atomic metadata mismatch", entry.Triple))
		}
		if entry.UnsupportedReason != tgt.UnsupportedReason {
			return failCheck("target metadata", fmt.Sprintf("%s unsupported_reason got %q want %q", entry.Triple, entry.UnsupportedReason, tgt.UnsupportedReason))
		}
		if entry.ExeExt != tgt.ExeExt {
			return failCheck("target metadata", fmt.Sprintf("%s exe_ext got %q want %q", entry.Triple, entry.ExeExt, tgt.ExeExt))
		}
		buildOnly := ctarget.IsBuildOnlyTarget(entry.Triple)
		if entry.BuildOnly != buildOnly {
			return failCheck("target metadata", fmt.Sprintf("%s build_only got %v want %v", entry.Triple, entry.BuildOnly, buildOnly))
		}
		if err := validateDoctorTargetRunMetadata(entry, tgt, buildOnly); err != nil {
			return failCheck("target metadata", err.Error())
		}
		if buildOnly {
			buildOnlyCount++
		}
	}
	wantCount := len(ctarget.SupportedTriples()) + len(ctarget.BuildOnlyTriples()) + len(ctarget.PlannedTriples())
	if len(entries) != wantCount {
		return failCheck("target metadata", fmt.Sprintf("got %d targets want %d", len(entries), wantCount))
	}
	return passCheck("target metadata", fmt.Sprintf("%d targets, %d build-only", len(entries), buildOnlyCount))
}

func validateDoctorTargetRunMetadata(entry targetReportEntry, tgt ctarget.Target, buildOnly bool) error {
	if entry.RunMode != tgt.RunMode.String() {
		return fmt.Errorf("%s run_mode got %q want %q", entry.Triple, entry.RunMode, tgt.RunMode.String())
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
			return fmt.Errorf("%s run_unsupported_reason is required when run_supported is false", entry.Triple)
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
				return fmt.Errorf("%s run_unsupported_reason got %q want empty when host probe is supported", entry.Triple, entry.RunUnsupportedReason)
			}
			return nil
		}
		if !strings.Contains(entry.RunUnsupportedReason, "no host fallback") {
			return fmt.Errorf("%s run_unsupported_reason must explain host probe failure and no fallback", entry.Triple)
		}
	case ctarget.RunModeWASIRunner:
		if entry.Triple != "wasm32-wasi" || buildOnly {
			return fmt.Errorf("%s wasi_runner mode is only valid for wasm32-wasi supported target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner != "wasmtime" && entry.RunRunner != "node-wasi" {
				return fmt.Errorf("%s run_runner got %q want wasmtime or node-wasi", entry.Triple, entry.RunRunner)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("%s run_unsupported_reason got %q want empty when runner is available", entry.Triple, entry.RunUnsupportedReason)
			}
			return nil
		}
		if entry.RunRunner != "" {
			return fmt.Errorf("%s run_runner got %q want empty when runner is unavailable", entry.Triple, entry.RunRunner)
		}
		if !strings.Contains(entry.RunUnsupportedReason, "missing WASI runner") {
			return fmt.Errorf("%s run_unsupported_reason must explain missing WASI runner", entry.Triple)
		}
	case ctarget.RunModeWebRunner:
		if entry.Triple != "wasm32-web" || buildOnly {
			return fmt.Errorf("%s web_runner mode is only valid for wasm32-web supported target", entry.Triple)
		}
		if entry.RunSupported {
			if entry.RunRunner == "" {
				return fmt.Errorf("%s run_runner is required when web runner is available", entry.Triple)
			}
			if entry.RunUnsupportedReason != "" {
				return fmt.Errorf("%s run_unsupported_reason got %q want empty when web runner is available", entry.Triple, entry.RunUnsupportedReason)
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
			return fmt.Errorf("%s run_unsupported_reason must explain missing web runner", entry.Triple)
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
			return fmt.Errorf("%s run_unsupported_reason is required for unsupported run mode", entry.Triple)
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
	commands := []string{"check", "build", "run", "fmt", "test", "doc", "interface", "smoke", "surface", "targets", "formats", "doctor", "actor-net", "project", "new", "lsp", "eco", "clean", "version"}
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
