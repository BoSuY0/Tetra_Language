package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

var commandLookPath = exec.LookPath
var webRunnerProbe = probeWebRunner

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

const supportedTargetsHelp = "linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-web"

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printUsage(stdout)
		return 0
	}
	switch args[0] {
	case "version":
		if len(args) > 1 {
			if isHelpArgs(args[1:]) {
				fmt.Fprintln(stdout, "usage: tetra version")
				return 0
			}
			fmt.Fprintln(stderr, "version does not accept arguments")
			return 2
		}
		fmt.Fprintln(stdout, compiler.Version())
		return 0
	case "targets":
		return runTargets(args[1:], stdout, stderr)
	case "features":
		return runFeatures(args[1:], stdout, stderr)
	case "formats":
		return runFormats(args[1:], stdout, stderr)
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "project":
		return runProject(args[1:], stdout, stderr)
	case "workspace":
		return runWorkspace(args[1:], stdout, stderr)
	case "interface":
		return runInterface(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "actor-net":
		return runActorNet(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "smoke":
		return runSmoke(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	case "test":
		return runTest(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "clean":
		return runClean(args[1:], stdout, stderr)
	case "eco":
		return runEco(args[1:], stdout, stderr)
	case "lsp":
		return runLSP(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func isHelpArgs(args []string) bool {
	return len(args) == 1 && (args[0] == "-h" || args[0] == "--help")
}

func runBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host")
	out := fs.String("o", "", "output path")
	allTargets := fs.Bool("all-targets", false, "build every target listed in Capsule.t4")
	interfaceOnly := fs.Bool("interface-only", false, "type-check interface/API graph without emitting executable code")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	artifactsMode := fs.String("artifacts", "strict", "artifact handling: strict or auto")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *artifactsMode != "strict" && *artifactsMode != "auto" {
		writeValidationDiagnostic(stderr, *diagnostics, "build --artifacts must be strict or auto")
		return 2
	}

	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "build accepts at most one input path")
		return 2
	}

	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *artifactsMode == "auto" && projectCtx != nil && projectCtx.Found {
		if err := buildCapsuleArtifacts(projectCtx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     *target,
			LockPath:   projectCtx.LockPath,
			Jobs:       *jobs,
			AllTargets: *allTargets,
		}); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Artifacts repaired: %s\n", projectCtx.CapsulePath)
		input, worldOpt, projectCtx, err = resolveCLIInput(input)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
	}

	if *allTargets {
		targets := projectBuildTargets(projectCtx)
		if len(targets) == 0 {
			writeValidationDiagnostic(stderr, *diagnostics, "build --all-targets requires targets in Capsule.t4")
			return 2
		}
		targetLinkObjects, err := projectLinkObjects(projectCtx, "", []string(linkObjects))
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		opt, err := buildOptions(*emit, *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 2
		}
		opt.ProjectRoot = worldOpt.Root
		opt.SourceRoots = worldOpt.SourceRoots
		opt.DependencyRoots = worldOpt.DependencyRoots
		opt.InterfaceOnly = *interfaceOnly
		for _, rawTarget := range targets {
			tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
			if !ok {
				return 2
			}
			if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
			if err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			opt.LinkObjectPaths = targetLinkObjects
			output := allTargetsOutput(*out, tgt, *emit)
			if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if *interfaceOnly {
				fmt.Fprintf(stdout, "Interface-only build checked: %s (%s)\n", input, tgt.Triple)
			} else {
				fmt.Fprintf(stdout, "Built: %s\n", output)
			}
		}
		return 0
	}

	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	output := *out
	if output == "" {
		output = defaultOutput(tgt, *emit)
	}

	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}

	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions(*emit, *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	opt.InterfaceOnly = *interfaceOnly
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *interfaceOnly {
		fmt.Fprintf(stdout, "Interface-only build checked: %s\n", input)
	} else {
		fmt.Fprintf(stdout, "Built: %s\n", output)
	}
	return 0
}

func runRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host")
	out := fs.String("o", "", "output path")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "run accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	isWASI := tgt.Triple == "wasm32-wasi"
	isWeb := tgt.Triple == "wasm32-web"
	if ctarget.IsBuildOnlyTarget(tgt.Triple) && !isWASI {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run target %s: build-only target emits artifacts only; unsupported runtime execution because the CLI does not provide a production runtime runner", tgt.Triple))
		return 2
	}
	if !isWASI && !isWeb {
		host, ok := hostTarget()
		if !ok || host != tgt.Triple {
			writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
			return 2
		}
	}
	output := *out
	tmpDir := ""
	if output == "" {
		var err error
		tmpDir, err = os.MkdirTemp("", "tetra-run-*")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		defer os.RemoveAll(tmpDir)
		output = filepath.Join(tmpDir, defaultOutput(tgt, "exe"))
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, *runtimeObject, targetLinkObjects, *jobs)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if isWASI {
		exit, err := execWASMProgram(output, stdout, stderr)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		return exit
	}
	if isWeb {
		exit, err := execWebProgram(output, stdout, stderr)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		return exit
	}
	return execProgram(output, stdout, stderr)
}

func runClean(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "remove cache entries for one target")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "clean does not accept positional arguments")
		return 2
	}
	if *target != "" {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		for _, path := range []string{
			filepath.Join(".tetra_cache", tgt.Triple),
			filepath.Join("tetra_cache", tgt.Triple),
		} {
			if err := os.RemoveAll(path); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		fmt.Fprintf(stdout, "Cleaned Tetra cache for %s\n", tgt.Triple)
		return 0
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "Cleaned Tetra cache")
	return 0
}

func buildOptions(emit string, runtimeMode string, islandsDebug bool, runtimeObject string, linkObjects []string, jobs int) (compiler.BuildOptions, error) {
	opt := compiler.BuildOptions{
		Jobs:              jobs,
		IslandsDebug:      islandsDebug,
		RuntimeObjectPath: runtimeObject,
		LinkObjectPaths:   linkObjects,
	}
	switch emit {
	case "", "exe":
		opt.Emit = compiler.EmitExe
	case "object":
		opt.Emit = compiler.EmitObject
	case "library":
		opt.Emit = compiler.EmitLibrary
	default:
		return opt, fmt.Errorf("unsupported --emit %q", emit)
	}
	switch runtimeMode {
	case "", "auto":
		opt.Runtime = compiler.RuntimeAuto
	case "selfhost":
		opt.Runtime = compiler.RuntimeSelfHost
	case "builtin":
		opt.Runtime = compiler.RuntimeBuiltin
	default:
		return opt, fmt.Errorf("unsupported --runtime %q", runtimeMode)
	}
	return opt, nil
}

func defaultTarget() string {
	if target, ok := hostTarget(); ok {
		return target
	}
	return "linux-x64"
}

func hostTarget() (string, bool) {
	tgt, ok := ctarget.Host()
	if !ok {
		return "", false
	}
	return tgt.Triple, true
}

func defaultOutput(tgt ctarget.Target, emit string) string {
	switch emit {
	case "object", "library":
		return "app.tobj"
	default:
		return "app" + tgt.ExeExt
	}
}

func projectDefaultTarget(ctx *cliProjectContext) string {
	targets := projectBuildTargets(ctx)
	if len(targets) > 0 {
		return targets[0]
	}
	return defaultTarget()
}

func projectBuildTargets(ctx *cliProjectContext) []string {
	if ctx == nil || !ctx.Found || len(ctx.Manifest.Targets) == 0 {
		return nil
	}
	return append([]string(nil), ctx.Manifest.Targets...)
}

func allTargetsOutput(out string, tgt ctarget.Target, emit string) string {
	name := "app-" + tgt.Triple
	if emit == "object" || emit == "library" {
		name += ".tobj"
	} else {
		name += tgt.ExeExt
	}
	if out == "" {
		return name
	}
	if strings.HasSuffix(out, string(os.PathSeparator)) {
		return filepath.Join(out, name)
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, name)
	}
	return filepath.Join(out, name)
}

func execProgram(path string, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type wasiRunner struct {
	Name   string
	Path   string
	Helper string
}

func discoverWASIRunner(repoRoot string) (wasiRunner, error) {
	if runner, err := commandLookPath("wasmtime"); err == nil {
		return wasiRunner{Name: "wasmtime", Path: runner}, nil
	}
	node, err := commandLookPath("node")
	if err != nil {
		return wasiRunner{}, fmt.Errorf("cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node")
	}
	if repoRoot == "" {
		if root, rootErr := findRepoRoot(); rootErr == nil {
			repoRoot = root
		}
	}
	helper := filepath.Join(repoRoot, "scripts", "tools", "wasi_run_module.mjs")
	if repoRoot == "" || !fileExists(helper) {
		return wasiRunner{}, fmt.Errorf("cannot run target wasm32-wasi: missing WASI node helper scripts/tools/wasi_run_module.mjs")
	}
	return wasiRunner{Name: "node-wasi", Path: node, Helper: helper}, nil
}

func discoverWebRunner() (string, error) {
	var probeFailure string
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		if runner, err := commandLookPath(candidate); err == nil {
			if err := webRunnerProbe(runner); err != nil {
				probeFailure = fmt.Sprintf("%s failed headless probe: %v", runner, err)
				continue
			}
			return runner, nil
		}
	}
	if probeFailure != "" {
		return "", fmt.Errorf("cannot run target wasm32-web: browser runner unavailable: %s", probeFailure)
	}
	return "", fmt.Errorf("cannot run target wasm32-web: browser runner unavailable; searched: chromium, chromium-browser, google-chrome, chrome")
}

type webRuntimeRunner struct {
	Name   string
	Path   string
	Helper string
}

func discoverWebRuntimeRunner(repoRoot string) (webRuntimeRunner, error) {
	node, err := commandLookPath("node")
	if err != nil {
		return webRuntimeRunner{}, fmt.Errorf("cannot run target wasm32-web: missing web runtime runner: need node")
	}
	if repoRoot == "" {
		if root, rootErr := findRepoRoot(); rootErr == nil {
			repoRoot = root
		}
	}
	helper := filepath.Join(repoRoot, "scripts", "tools", "web_run_module.mjs")
	if repoRoot == "" || !fileExists(helper) {
		return webRuntimeRunner{}, fmt.Errorf("cannot run target wasm32-web: missing web node helper scripts/tools/web_run_module.mjs")
	}
	return webRuntimeRunner{Name: "node-web", Path: node, Helper: helper}, nil
}

func probeWebRunner(runner string) error {
	cmd := exec.Command(
		runner,
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--dump-dom",
		"about:blank",
	)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func execWASMProgram(path string, stdout io.Writer, stderr io.Writer) (int, error) {
	runner, err := discoverWASIRunner("")
	if err != nil {
		return 0, err
	}
	return execWASMProgramWithRunner(path, runner, stdout, stderr)
}

func execWASMProgramWithRunner(path string, runner wasiRunner, stdout io.Writer, stderr io.Writer) (int, error) {
	var cmd *exec.Cmd
	switch runner.Name {
	case "wasmtime":
		cmd = exec.Command(runner.Path, "run", path)
	case "node-wasi":
		cmd = exec.Command(runner.Path, runner.Helper, path)
	default:
		return 0, fmt.Errorf("unsupported WASI runner %q", runner.Name)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	if cmd.ProcessState == nil {
		return 0, nil
	}
	return cmd.ProcessState.ExitCode(), nil
}

func execWebProgram(path string, stdout io.Writer, stderr io.Writer) (int, error) {
	runner, err := discoverWebRuntimeRunner("")
	if err != nil {
		return 0, err
	}
	return execWebProgramWithRunner(path, runner, stdout, stderr)
}

func execWebProgramWithRunner(path string, runner webRuntimeRunner, stdout io.Writer, stderr io.Writer) (int, error) {
	cmd := exec.Command(runner.Path, runner.Helper, path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	if cmd.ProcessState == nil {
		return 0, nil
	}
	return cmd.ProcessState.ExitCode(), nil
}

func execWebProgramWithBrowserRunner(path string, runner string, stdout io.Writer, stderr io.Writer) (int, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	dir := filepath.Dir(absPath)
	wasmFile := filepath.Base(absPath)
	loaderFile := strings.TrimSuffix(wasmFile, filepath.Ext(wasmFile)) + ".mjs"
	if !fileExists(filepath.Join(dir, loaderFile)) {
		return 0, fmt.Errorf("cannot run target wasm32-web: missing web loader %s", filepath.Join(dir, loaderFile))
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, err
	}
	runnerHTML := filepath.Join(dir, ".tetra-web-runner.html")
	html := fmt.Sprintf(`<!doctype html>
<html>
  <body>
    <pre id="result">pending</pre>
    <script type="module">
      import { runTetra } from './%s';
      const el = document.getElementById('result');
      try {
        const code = await runTetra(new URL('./%s', import.meta.url));
        el.textContent = 'exit:' + (code | 0);
      } catch (err) {
        el.textContent = 'error:' + String(err && err.message ? err.message : err);
      }
    </script>
  </body>
</html>
`, loaderFile, wasmFile)
	if err := os.WriteFile(runnerHTML, []byte(html), 0o644); err != nil {
		return 0, err
	}
	defer os.Remove(runnerHTML)

	server := exec.Command("python3", "-m", "http.server", strconv.Itoa(port), "--bind", "127.0.0.1", "--directory", dir)
	server.Stdout = io.Discard
	server.Stderr = io.Discard
	if err := server.Start(); err != nil {
		return 0, fmt.Errorf("cannot run target wasm32-web: start local HTTP server: %w", err)
	}
	defer func() {
		if server.Process != nil {
			_ = server.Process.Kill()
			_, _ = server.Process.Wait()
		}
	}()
	time.Sleep(300 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/%s", port, filepath.Base(runnerHTML))
	cmd := exec.Command(runner, "--headless", "--disable-gpu", "--no-sandbox", "--virtual-time-budget=12000", "--dump-dom", url)
	var dom bytes.Buffer
	cmd.Stdout = &dom
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("cannot run target wasm32-web: browser runner failed: %w", err)
	}
	result := extractWebRunnerResult(dom.String())
	if strings.HasPrefix(result, "exit:") {
		code, err := strconv.Atoi(strings.TrimPrefix(result, "exit:"))
		if err != nil {
			return 0, fmt.Errorf("cannot run target wasm32-web: invalid browser exit result %q", result)
		}
		return code & 0xff, nil
	}
	if strings.HasPrefix(result, "error:") {
		return 1, fmt.Errorf("cannot run target wasm32-web: %s", strings.TrimPrefix(result, "error:"))
	}
	return 1, fmt.Errorf("cannot run target wasm32-web: browser did not report an exit result")
}

func extractWebRunnerResult(dom string) string {
	re := regexp.MustCompile(`<pre id="result">([^<]*)</pre>`)
	m := re.FindStringSubmatch(dom)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "examples")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeDiagnostic(w io.Writer, mode string, err error) {
	if mode == "json" {
		writeDiagnosticObject(w, compiler.DiagnosticFromError(err))
		return
	}
	fmt.Fprintln(w, err)
}

func writeDiagnosticWithHint(w io.Writer, mode string, message string, hint string) {
	if mode == "json" {
		writeDiagnosticObject(w, compiler.Diagnostic{
			Code:     compiler.DiagnosticCodeParse,
			Message:  message,
			Severity: "error",
			Hint:     hint,
		})
		return
	}
	fmt.Fprintln(w, message)
}

func parseBuildTargetOrReport(rawTarget string, diagnosticsMode string, stderr io.Writer) (ctarget.Target, bool) {
	tgt, err := ctarget.Parse(rawTarget)
	if err == nil {
		return tgt, true
	}
	if targetErr, ok := err.(ctarget.UnsupportedTargetError); ok {
		msg := fmt.Sprintf(
			"unsupported target: %s; supported targets: %s; build-only targets: %s",
			targetErr.Triple,
			strings.Join(ctarget.SupportedTriples(), ", "),
			strings.Join(ctarget.BuildOnlyTriples(), ", "),
		)
		writeDiagnosticWithHint(stderr, diagnosticsMode, msg, "run `tetra targets` to list valid targets")
		return ctarget.Target{}, false
	}
	writeDiagnostic(stderr, diagnosticsMode, err)
	return ctarget.Target{}, false
}

func writeValidationDiagnostic(w io.Writer, mode string, message string) {
	writeDiagnostic(w, mode, fmt.Errorf("%s", message))
}

func validateDiagnosticsMode(w io.Writer, mode string) bool {
	if mode == "text" || mode == "json" {
		return true
	}
	fmt.Fprintln(w, "unsupported --diagnostics format")
	return false
}

func writeDiagnosticObject(w io.Writer, diag compiler.Diagnostic) {
	raw, err := json.Marshal(diag)
	if err == nil {
		fmt.Fprintln(w, string(raw))
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra <version|targets|features|formats|doctor|actor-net|project|workspace|new|check|build|run|smoke|fmt|test|doc|interface|clean|eco|lsp> [options]")
}
