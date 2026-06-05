package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

func TestSurfaceRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredSurfaceRuntimeSymbols()
	want := []string{
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("surface runtime symbols = %#v, want %#v", got, want)
	}
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_surface_open", params: 4, rets: 1},
		{name: "__tetra_surface_close", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_kind", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_x", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_y", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_button", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_into", params: 3, rets: 1},
		{name: "__tetra_surface_poll_event_text_len", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_text_into", params: 3, rets: 1},
		{name: "__tetra_surface_clipboard_write_text", params: 3, rets: 1},
		{name: "__tetra_surface_clipboard_read_text_into", params: 3, rets: 1},
		{name: "__tetra_surface_poll_composition_into", params: 3, rets: 1},
		{name: "__tetra_surface_begin_frame", params: 1, rets: 1},
		{name: "__tetra_surface_present_rgba", params: 6, rets: 1},
		{name: "__tetra_surface_now_ms", params: 0, rets: 1},
		{name: "__tetra_surface_request_redraw", params: 1, rets: 1},
	}
	for _, tt := range tests {
		sig, ok := runtimeObjectSignature(tt.name)
		if !ok {
			t.Fatalf("missing runtime signature for %s", tt.name)
		}
		if sig.paramSlots != tt.params || sig.returnSlots != tt.rets {
			t.Fatalf("%s signature = params %d returns %d, want params %d returns %d", tt.name, sig.paramSlots, sig.returnSlots, tt.params, tt.rets)
		}
	}
}

func TestCollectSurfaceRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectSurfaceRuntimeUsage(checked) {
		t.Fatalf("surface runtime usage was not collected")
	}
}

func TestValidateSurfaceRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithSurfaceRuntimeSignatures()
	if err := validateSurfaceRuntimeObject(obj); err != nil {
		t.Fatalf("validate surface runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_surface_present_rgba", 5, 1)
	err := validateSurfaceRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected surface runtime signature mismatch")
	}
	if !strings.Contains(err.Error(), "runtime object symbol '__tetra_surface_present_rgba' signature mismatch") ||
		!strings.Contains(err.Error(), "params=5 want=6") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingSurfaceSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if tgt.Triple != "linux-x64" {
		t.Skipf("surface runtime is linux-x64 only for this slice, host is %s", tgt.Triple)
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_surface.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_surface",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "surface_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "surface_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected missing surface runtime symbol failure")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_surface_open'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSurfaceRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "surface_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
		{target: "x32", want: "linux-x32"},
		{target: "x86", want: "linux-x86"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "surface-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported surface runtime diagnostic")
			}
			want := "surface runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestSurfaceRuntimeBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("demo", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let present: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let first_close: Int = core.surface_close(handle)
    let second_close: Int = core.surface_close(handle)
    if handle > 2 && present == 0 && first_close == 0 && second_close != 0:
        return 42
    return 1
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want kernel-backed linux-x64 Surface host result 42", exitCode)
	}
}

func TestSurfaceRuntimePollEventCoordinatesLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface:
    let handle: Int = core.surface_open("event-probe", 320, 200)
    let kind: Int = core.surface_poll_event_kind(handle)
    let x: Int = core.surface_poll_event_x(handle)
    let y: Int = core.surface_poll_event_y(handle)
    let button: Int = core.surface_poll_event_button(handle)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && kind == 5 && x == 48 && y == 96 && button == 1:
        return 42
    return kind
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want host-provided linux-x64 Surface pointer event result 42", exitCode)
	}
}

func TestSurfaceRuntimePollEventTextPayloadLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("text-probe", 320, 200)
    var text: []u8 = core.make_u8(4)
    let text_len: Int = core.surface_poll_event_text_len(handle)
    let copied: Int = core.surface_poll_event_text_into(handle, text)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && text_len == 2 && copied == 2 && text[0] == 79 && text[1] == 75:
        return 42
    return text_len + copied
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want host-provided linux-x64 Surface text payload result 42", exitCode)
	}
}

func TestSurfaceRuntimePollEventBufferLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-buffer-probe", 320, 200)
    var event: []i32 = core.make_i32(9)
    let copied: Int = core.surface_poll_event_into(handle, event)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied == 9 && event[0] == 5 && event[1] == 48 && event[2] == 96 && event[3] == 1 && event[4] == 0 && event[5] == 320 && event[6] == 200 && event[7] == 0 && event[8] == 0:
        return 42
    return copied
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want host-provided linux-x64 Surface event buffer result 42", exitCode)
	}
}

func TestSurfaceRuntimePollEventBufferSequenceLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-sequence-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var third: []i32 = core.make_i32(9)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let copied3: Int = core.surface_poll_event_into(handle, third)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied1 == 9 && first[0] == 5 && first[1] == 48 && first[2] == 96 && first[3] == 1 && first[4] == 0 && first[5] == 320 && first[6] == 200 && first[7] == 0 && first[8] == 0 && copied2 == 9 && second[0] == 6 && second[1] == 0 && second[2] == 0 && second[3] == 0 && second[4] == 32 && second[5] == 320 && second[6] == 200 && second[7] == 1 && second[8] == 0 && copied3 == 9 && third[0] == 2 && third[1] == 0 && third[2] == 0 && third[3] == 0 && third[4] == 0 && third[5] == 400 && third[6] == 240 && third[7] == 2 && third[8] == 0:
        return 42
    return copied1 + copied2 + copied3
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want linux-x64 Surface poll_event_into sequence pointer/key/resize", exitCode)
	}
}

func TestSurfaceRuntimePresentPreservesPollEventCursorLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-present-cursor-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var pixels: []u8 = core.make_u8(16)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let presented: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && presented == 0 && copied1 == 9 && first[0] == 5 && copied2 == 9 && second[0] == 6 && second[4] == 32:
        return 42
    return second[0]
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want present_rgba to preserve linux-x64 Surface event cursor", exitCode)
	}
}

func TestSurfaceCounterExampleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-counter")
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "surface_counter.tetra"), outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build surface counter: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want deterministic Surface counter result 1", exitCode)
	}
}

func TestSurfaceTextInputExampleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-text-input")
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "surface_text_input.tetra"), outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build surface text input: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want TextBox-owned host text buffer result 42", exitCode)
	}
}

func TestSurfaceMigrationExamplesBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	cases := []struct {
		name         string
		src          string
		expectedExit int
	}{
		{name: "ui_web_smoke", src: filepath.Join("..", "examples", "surface_migration_ui_web_smoke.tetra"), expectedExit: 2},
		{name: "ui_native_shell_smoke", src: filepath.Join("..", "examples", "surface_migration_ui_native_shell_smoke.tetra"), expectedExit: 11},
		{name: "dogfood_web_ui", src: filepath.Join("..", "examples", "surface_migration_dogfood_web_ui.tetra"), expectedExit: 3},
		{name: "tetra_control_center", src: filepath.Join("..", "examples", "surface_migration_tetra_control_center.tetra"), expectedExit: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			outPath := filepath.Join(tmp, tc.name)
			if _, err := BuildFileWithStatsOpt(tc.src, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
				t.Fatalf("build %s: %v", tc.src, err)
			}
			if err := verifyELF(outPath); err != nil {
				t.Fatalf("verify ELF: %v", err)
			}
			stdout, exitCode := runBinary(t, outPath)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tc.expectedExit {
				t.Fatalf("exit code = %d, want deterministic Surface migration result %d", exitCode, tc.expectedExit)
			}
		})
	}
}

func TestSurfaceCounterExampleBuildWASM32WebSurfaceHost(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-counter.wasm")
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "surface_counter.tetra"), outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build surface counter wasm32-web: %v", err)
	}
	if raw, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("read wasm output: %v", err)
	} else if len(raw) < 8 || string(raw[:4]) != "\x00asm" {
		header := raw
		if len(header) > 4 {
			header = header[:4]
		}
		t.Fatalf("wasm output has invalid header: % x", header)
	}

	loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
	loader, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read wasm Surface loader: %v", err)
	}
	for _, want := range []string{
		"tetra_surface_host_v1",
		"createSurfaceHost(instanceRef)",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(string(loader), want) {
			t.Fatalf("wasm Surface loader missing %q:\n%s", want, loader)
		}
	}
	for _, sidecar := range []string{
		strings.TrimSuffix(outPath, ".wasm") + ".ui.json",
		strings.TrimSuffix(outPath, ".wasm") + ".ui.web.mjs",
		strings.TrimSuffix(outPath, ".wasm") + ".ui.html",
	} {
		if _, err := os.Stat(sidecar); err == nil {
			t.Fatalf("Surface wasm build must not emit legacy metadata UI sidecar %s", sidecar)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", sidecar, err)
		}
	}
}

func TestSurfaceTextInputExampleBuildWASM32WebSurfaceHost(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-text-input.wasm")
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "surface_text_input.tetra"), outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build surface text input wasm32-web: %v", err)
	}
	if raw, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("read wasm output: %v", err)
	} else if len(raw) < 8 || string(raw[:4]) != "\x00asm" {
		header := raw
		if len(header) > 4 {
			header = header[:4]
		}
		t.Fatalf("wasm output has invalid header: % x", header)
	}

	loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
	loader, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read wasm Surface loader: %v", err)
	}
	for _, want := range []string{
		"tetra_surface_host_v1",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(string(loader), want) {
			t.Fatalf("wasm Surface text loader missing %q:\n%s", want, loader)
		}
	}
}

func TestTenSlotReturnDoesNotClobberBuiltinRuntimeSchedulerLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
struct Ten:
    a: Int
    b: Int
    c: Int
    d: Int
    e: Int
    f: Int
    g: Int
    h: Int
    i: Int
    j: Int

func make_ten() -> Ten:
    return Ten(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10)

func main() -> Int
uses runtime:
    let ten: Ten = make_ten()
    let _: Int = core.time_now_ms()
    return ten.a
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 10-slot return to preserve runtime scheduler state", exitCode)
	}
}

func runtimeObjectWithSurfaceRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredSurfaceRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing surface runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}
