package main

import (
	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestTargetsCommandText(t *testing.T) {
	var stdout bytes.Buffer
	code := runCLI([]string{"targets"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("targets exit code = %d, stdout=%q", code, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{"Supported targets:", "linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web", "Build-only targets:", "linux-x86", "linux-x32", "Planned targets:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("targets output missing %q:\n%s", want, out)
		}
	}
}

func TestTargetsCommandJSON(t *testing.T) {
	type targetMeta struct {
		Triple                  string `json:"triple"`
		Status                  string `json:"status"`
		OS                      string `json:"os"`
		Arch                    string `json:"arch"`
		ABI                     string `json:"abi"`
		DataModel               string `json:"data_model"`
		Format                  string `json:"format"`
		ExeExt                  string `json:"exe_ext"`
		BuildOnly               bool   `json:"build_only"`
		RunMode                 string `json:"run_mode"`
		RunRunner               string `json:"run_runner,omitempty"`
		RunSupported            bool   `json:"run_supported"`
		RunUnsupportedReason    string `json:"run_unsupported_reason"`
		PointerWidthBits        int    `json:"pointer_width_bits"`
		RegisterWidthBits       int    `json:"register_width_bits"`
		NativeIntWidthBits      int    `json:"native_int_width_bits"`
		Endian                  string `json:"endian"`
		StackAlignmentBytes     int    `json:"stack_alignment_bytes"`
		MaxAtomicWidthBits      int    `json:"max_atomic_width_bits"`
		AtomicWidthBits         []int  `json:"atomic_width_bits"`
		AtomicPointerWidthBits  int    `json:"atomic_pointer_width_bits"`
		UnsupportedReason       string `json:"unsupported_reason"`
		SupportsDebugInfo       bool   `json:"supports_debug_info"`
		SupportsReleaseOptimize bool   `json:"supports_release_optimize"`
	}
	var report struct {
		Supported []string     `json:"supported"`
		BuildOnly []string     `json:"build_only"`
		Planned   []string     `json:"planned"`
		Targets   []targetMeta `json:"targets"`
	}
	runCLIJSONStdout(t, []string{"targets", "--format=json"}, 0, &report)
	if strings.Join(report.Supported, ",") != "linux-x64,windows-x64,macos-x64,wasm32-wasi,wasm32-web" {
		t.Fatalf("supported targets = %#v", report.Supported)
	}
	if strings.Join(report.BuildOnly, ",") != "linux-x86,linux-x32" {
		t.Fatalf("build-only targets = %#v", report.BuildOnly)
	}
	if len(report.Planned) != 0 {
		t.Fatalf("planned targets = %#v", report.Planned)
	}
	if len(report.Targets) != 7 {
		t.Fatalf("targets metadata count = %d, want 7: %#v", len(report.Targets), report.Targets)
	}
	byTriple := map[string]targetMeta{}
	for _, tgt := range report.Targets {
		if byTriple[tgt.Triple].Triple != "" {
			t.Fatalf("duplicate target metadata for %s in %#v", tgt.Triple, report.Targets)
		}
		byTriple[tgt.Triple] = tgt
	}
	for _, triple := range append(append([]string{}, report.Supported...), report.BuildOnly...) {
		if byTriple[triple].Triple == "" {
			t.Fatalf("target metadata missing %s in %#v", triple, report.Targets)
		}
	}
	if got := byTriple["linux-x64"]; got.Status != "supported" || got.OS != "linux" || got.Arch != "x64" || got.ABI != "sysv" || got.DataModel != "lp64" || got.Format != "elf" || got.PointerWidthBits != 64 || got.RegisterWidthBits != 64 || got.NativeIntWidthBits != 64 || got.AtomicPointerWidthBits != 64 || !reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32, 64}) || got.Endian != "little" || got.BuildOnly || !got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
		t.Fatalf("linux-x64 metadata = %#v", got)
	}
	if got := byTriple["windows-x64"]; got.Status != "supported" || got.OS != "windows" || got.ABI != "win64" || got.DataModel != "llp64" || got.Format != "pe" || got.ExeExt != ".exe" || got.PointerWidthBits != 64 || got.RegisterWidthBits != 64 || !got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
		t.Fatalf("windows-x64 metadata = %#v", got)
	}
	if got := byTriple["linux-x86"]; got.Status != "build_only" || got.OS != "linux" || got.Arch != "x86" || got.ABI != "i386-sysv" || got.DataModel != "ilp32" || got.PointerWidthBits != 32 || got.RegisterWidthBits != 32 || got.NativeIntWidthBits != 32 || got.AtomicPointerWidthBits != 32 || got.MaxAtomicWidthBits != 32 || !reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32}) || got.RunMode != "host_probed" || !got.BuildOnly || !strings.Contains(got.UnsupportedReason, "not implemented yet") || !strings.Contains(got.UnsupportedReason, "executable build/link") || !strings.Contains(got.UnsupportedReason, "run/test execution") || !strings.Contains(got.UnsupportedReason, "stdout write/string literal data") || !strings.Contains(got.UnsupportedReason, "stack-argument") || !strings.Contains(got.UnsupportedReason, "scalar global") || !strings.Contains(got.UnsupportedReason, "symbol-backed callback") || !strings.Contains(got.UnsupportedReason, "heap-backed slice allocation/indexing") || !strings.Contains(got.UnsupportedReason, "raw ptr_add/load/store") || !strings.Contains(got.UnsupportedReason, "MMIO read/write") || !strings.Contains(got.UnsupportedReason, "scoped island bump allocation/free") || !strings.Contains(got.UnsupportedReason, "debug double-free guard/page-protect") {
		t.Fatalf("linux-x86 metadata = %#v", got)
	} else {
		requireUnsupportedReasonContains(t, "linux-x86", got.UnsupportedReason, []string{
			"i386 SysV ABI classifier",
			"explicit filesystem/networking stdlib plus time/task/actors target-runtime boundary diagnostics",
			"x86 pointer/native-libc/function-pointer @export diagnostics",
			"source native scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"source-level atomic diagnostics",
		})
		if got.RunSupported {
			if got.RunUnsupportedReason != "" {
				t.Fatalf("linux-x86 supported host-probed metadata = %#v", got)
			}
		} else if !strings.Contains(got.RunUnsupportedReason, "host does not support Linux i386 execution") || !strings.Contains(got.RunUnsupportedReason, "no host fallback") {
			t.Fatalf("linux-x86 unsupported host-probed metadata = %#v", got)
		}
	}
	if got := byTriple["linux-x32"]; got.Status != "build_only" || got.OS != "linux" || got.Arch != "x64" || got.ABI != "x32-sysv" || got.DataModel != "x32" || got.PointerWidthBits != 32 || got.RegisterWidthBits != 64 || got.NativeIntWidthBits != 32 || got.AtomicPointerWidthBits != 32 || !reflect.DeepEqual(got.AtomicWidthBits, []int{8, 16, 32, 64}) || got.RunMode != "host_probed" || !got.BuildOnly || !strings.Contains(got.UnsupportedReason, "full linux-x32 runtime/stdlib/FFI support is not implemented yet") || !strings.Contains(got.UnsupportedReason, "executable build/link") || !strings.Contains(got.UnsupportedReason, "object codegen") || !strings.Contains(got.UnsupportedReason, "self-host runtime builds") || !strings.Contains(got.UnsupportedReason, "compiler-owned target suites") || !strings.Contains(got.UnsupportedReason, "host-probed source run/test execution") || !strings.Contains(got.UnsupportedReason, "Linux kernel supports the x32 ABI") {
		t.Fatalf("linux-x32 metadata = %#v", got)
	} else {
		requireUnsupportedReasonContains(t, "linux-x32", got.UnsupportedReason, []string{
			"x32 SysV ABI classifier",
			"raw ptr_add/load/store",
			"pointer load/store",
			"MMIO read/write",
			"scoped island bump allocation/free",
			"explicit filesystem/networking stdlib plus x32 multi-spawn actors/task, task-group, and typed-task runtime boundary diagnostics",
			"x32 pointer/native-libc/function-pointer @export diagnostics",
			"source native scalar diagnostics",
			"pointer-only atomic ABI-width object check",
			"dword pointer atomics",
			"x32 syscall numbers",
		})
		if got.RunSupported {
			if got.RunUnsupportedReason != "" {
				t.Fatalf("linux-x32 supported host-probed metadata = %#v", got)
			}
		} else if !strings.Contains(got.RunUnsupportedReason, "host does not support Linux x32 ABI execution") || !strings.Contains(got.RunUnsupportedReason, "no host fallback") {
			t.Fatalf("linux-x32 unsupported host-probed metadata = %#v", got)
		}
	}
	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		got := byTriple[triple]
		if got.Status != "supported" || got.Arch != "wasm32" || got.DataModel != "ilp32" || got.PointerWidthBits != 32 || got.Format != "wasm" || got.ExeExt != ".wasm" || got.BuildOnly || got.SupportsDebugInfo || !got.SupportsReleaseOptimize {
			t.Fatalf("%s metadata = %#v", triple, got)
		}
	}
	if got := byTriple["wasm32-wasi"]; got.RunMode != "wasi_runner" {
		t.Fatalf("wasm32-wasi runner metadata = %#v", got)
	}
	if got := byTriple["wasm32-web"]; got.RunMode != "web_runner" {
		t.Fatalf("wasm32-web runtime metadata = %#v", got)
	} else if got.RunSupported {
		if got.RunRunner == "" || got.RunUnsupportedReason != "" {
			t.Fatalf("wasm32-web supported runner metadata = %#v", got)
		}
	} else if got.RunRunner != "" || !strings.Contains(got.RunUnsupportedReason, "runner unavailable") {
		t.Fatalf("wasm32-web unsupported runner metadata = %#v", got)
	}
}

func TestTargetsCommandJSONMarksWASIRunSupportedWhenRunnerExists(t *testing.T) {
	restore := stubLookPath(func(name string) (string, error) {
		if name == "wasmtime" {
			return "/usr/bin/wasmtime", nil
		}
		if name == "chromium" {
			return "/usr/bin/chromium", nil
		}
		return "", exec.ErrNotFound
	})
	defer restore()

	report := targetsJSONForTest(t)
	wasm := targetMetaForTest(t, report, "wasm32-wasi")
	if wasm.BuildOnly || wasm.RunMode != "wasi_runner" || wasm.RunRunner != "wasmtime" || !wasm.RunSupported || wasm.RunUnsupportedReason != "" {
		t.Fatalf("wasm32-wasi metadata with runner = %#v", wasm)
	}
	web := targetMetaForTest(t, report, "wasm32-web")
	if web.BuildOnly || web.RunMode != "web_runner" || web.RunRunner != "/usr/bin/chromium" || !web.RunSupported || web.RunUnsupportedReason != "" {
		t.Fatalf("wasm32-web metadata with browser runner = %#v", web)
	}
}

func TestTargetsCommandJSONMarksWASIRunUnsupportedWhenRunnerMissing(t *testing.T) {
	restore := stubLookPath(func(name string) (string, error) {
		return "", exec.ErrNotFound
	})
	defer restore()

	report := targetsJSONForTest(t)
	wasm := targetMetaForTest(t, report, "wasm32-wasi")
	if wasm.BuildOnly || wasm.RunMode != "wasi_runner" || wasm.RunRunner != "" || wasm.RunSupported || !strings.Contains(wasm.RunUnsupportedReason, "missing WASI runner") {
		t.Fatalf("wasm32-wasi metadata without runner = %#v", wasm)
	}
}

type targetMetaJSONForTest struct {
	Triple               string `json:"triple"`
	BuildOnly            bool   `json:"build_only"`
	RunMode              string `json:"run_mode"`
	RunRunner            string `json:"run_runner,omitempty"`
	RunSupported         bool   `json:"run_supported"`
	RunUnsupportedReason string `json:"run_unsupported_reason"`
}

type targetsJSONReportForTest struct {
	Targets []targetMetaJSONForTest `json:"targets"`
}

func targetsJSONForTest(t *testing.T) targetsJSONReportForTest {
	t.Helper()
	var report targetsJSONReportForTest
	runCLIJSONStdout(t, []string{"targets", "--format=json"}, 0, &report)
	return report
}

func targetMetaForTest(t *testing.T, report targetsJSONReportForTest, triple string) targetMetaJSONForTest {
	t.Helper()
	for _, target := range report.Targets {
		if target.Triple == triple {
			return target
		}
	}
	t.Fatalf("missing target metadata for %s in %#v", triple, report.Targets)
	return targetMetaJSONForTest{}
}

func requireUnsupportedReasonContains(t *testing.T, triple string, reason string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(reason, want) {
			t.Fatalf("%s unsupported_reason missing %q: %q", triple, want, reason)
		}
	}
}

func TestTargetsCommandRejectsUnsupportedFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"targets", "--format=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("targets exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFeaturesCommandJSON(t *testing.T) {
	var report struct {
		Schema   string `json:"schema"`
		Version  string `json:"version"`
		Features []struct {
			ID        string   `json:"id"`
			Name      string   `json:"name"`
			Status    string   `json:"status"`
			Since     string   `json:"since"`
			Scope     string   `json:"scope"`
			Stability string   `json:"stability"`
			Docs      []string `json:"docs"`
		} `json:"features"`
	}
	runCLIJSONStdout(t, []string{"features", "--format=json"}, 0, &report)
	if report.Schema != "tetra.features.v1" {
		t.Fatalf("features schema = %q", report.Schema)
	}
	if report.Version != compiler.Version() {
		t.Fatalf("features version = %q, want %q", report.Version, compiler.Version())
	}
	statusByID := map[string]string{}
	statusSeen := map[string]bool{}
	for _, feature := range report.Features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" || feature.Stability == "" || len(feature.Docs) == 0 {
			t.Fatalf("feature missing required metadata: %#v", feature)
		}
		statusByID[feature.ID] = feature.Status
		statusSeen[feature.Status] = true
		if feature.ID == "language.enum-payload-match" {
			if feature.Status != "current" || feature.Since != "v0.3.0" {
				t.Fatalf("enum payload feature lifecycle = status %q since %q, want current since v0.3.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"positional enum payload constructors", "match/catch/if-let", "exhaustive unguarded enum match/catch", "nested destructuring patterns", "guard expansion remain future/post-v1"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("enum payload feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.protocol-bound-generics-static" {
			if feature.Status != "current" || feature.Since != "v0.3.0" {
				t.Fatalf("protocol-bound generics lifecycle = status %q since %q, want current since v0.3.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"validated statically during monomorphization", "same-module and cross-module impl conformance", "visibility diagnostics", "calling protocol requirements through generic bounds", "dynamic dispatch remain unsupported"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("protocol-bound generics feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.generics-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("generics MVP lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"statically monomorphized", "no runtime generic values or dynamic dispatch", "generic structs", "future/post-v1"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("generics MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.protocol-conformance-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("protocol conformance MVP lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"checked statically", "generic requirement signature shape", "no witness tables", "dynamic dispatch remain post-v1"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("protocol conformance MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.callable-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("callable MVP lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"Level 0 callable surface", "symbol-backed non-capturing callable paths", "full first-class function values remain out of scope"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.callable-level1" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("callable Level 1 lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production non-capturing symbol-backed callable Level 1", "function-typed locals, aliases, callbacks", "signature-compatible mutable local reassignment", "captured closure escape beyond the fnptr Level 2 slice", "full first-class function values remain out of scope"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable Level 1 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.ownership-markers-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("ownership markers MVP lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{
				"conservative borrow/inout/consume marker checks",
				"same-module/cross-module struct-field and enum-payload partial consume with whole-value call/let/return and enum wrapper-constructor rejection",
				"borrow escape diagnostics for scalar ptr including same-module/cross-module scalar ptr consume and inout assignment",
				"same-module/cross-module borrowed scalar ptr escapes through ptr-containing struct inout assignment",
				"same-module/cross-module fixed-array alias return plus direct global assignment, optional global assignment, and inout assignment escapes with stable TETRA2102 diagnostic evidence",
				"borrowed string alias return/global assignment escapes",
				"ptr/slice optional assignment return/owned/consume/inout escape",
				"same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence",
				"same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence",
				"same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence",
				"ptr optional assignment if-let/match global escape",
				"same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence",
				"function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence",
				"same-module/cross-module protocol parameter ownership matching plus same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence",
				"same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence",
				"use-after-consume",
				"not a full SSA lifetime solver",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("ownership markers MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.resource-lifetime-mvp" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("resource lifetime MVP lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{
				"conservative resource finalization checks",
				"task handles",
				"island handles",
				"stable ownership safety JSON diagnostics for resource use-after-free, double-join, and ambiguous-provenance cases",
				"same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module island whole-optional use-after-payload-free diagnostics",
				"double-use",
				"ambiguous provenance",
				"not a full SSA lifetime solver",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("resource lifetime MVP feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "actors.task-transfer-safety" {
			if feature.Status != "current" || feature.Since != "v0.2.0" {
				t.Fatalf("actor/task transfer safety lifecycle = status %q since %q, want current since v0.2.0", feature.Status, feature.Since)
			}
			for _, want := range []string{
				"conservative actor/task ownership transfer checks",
				"worker entrypoints",
				"branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence",
				"actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence",
				"island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence",
				"same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence",
				"same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence",
				"conservative local MVP",
				"distributed actors",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("actor/task transfer feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "actors.distributed-runtime" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("distributed actor runtime lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, wantDoc := range []string{"docs/spec/current_supported_surface.md", "docs/spec/actors.md", "docs/user/async_actors_guide.md"} {
				if !containsString(feature.Docs, wantDoc) {
					t.Fatalf("distributed actor runtime docs missing %s: %#v", wantDoc, feature)
				}
			}
			for _, want := range []string{
				"production Linux-x64 distributed actor runtime path",
				"actornet loopback TCP broker",
				"distributed node identity",
				"remote actor handles",
				"network mailbox send/receive",
				"i32, tagged, and typed frames",
				"missing-node failure/status propagation",
				"task cancel/join handles",
				"tetra.actors.distributed-runtime.v1 smoke evidence",
				"transport-only or fake reports",
				"non-Linux-x64 targets",
				"broader structured-concurrency guarantees",
			} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("distributed actor runtime feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.lifetime-ssa" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("lifetime SSA lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production SSA-like local lifetime join analysis", "ownership consume state", "resource finalization state", "optional region-wrapper escapes", "maybe-consumed diagnostics", "richer interprocedural lifetime proofs"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("lifetime SSA feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.callable-level2" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("callable Level 2 lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production captured closure Level 2 slice", "fnptr-backed function-typed locals", "function-typed returns", "immutable local struct fields or enum payloads", "larger immutable environments are promoted under language.full-first-class-callables"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("callable Level 2 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "language.full-first-class-callables" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("full first-class callable lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production first-class callable/function-value semantics", "fixed 4-slot callable handle", "larger immutable Int/Bool/String/simple-aggregate captures", "synchronous callback arguments", "cross-module returned values", "stable JSON diagnostics for mutable by-reference captures including callable mutable-capture global-escape", "callable mutable-capture heap-escape", "callable pointer/resource capture escape", "function-typed storage/return unsupported capture rejection", "captured callable/function-typed parameter global-storage escape", "unsupported function-value escape outside the fnptr ABI", "unsupported function-value call", "capturing closure raw-ptr escape", "captured closure explicit type-arg rejection", "function-typed explicit type-arg rejection", "generic closure capture and generic callback-closure capture rejection", "generic closure pointer/direct-call rejection", "imported mutable function-typed global boundary"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("full first-class callable feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "ui.metadata-v1" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("ui.metadata-v1 lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production UI metadata contract", "deterministic tetra.ui.v1 JSON", "browser-backed web command-dispatch runtime", "wasm32-web command dispatch", "post-v0.4 Web UI runtime smoke"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("ui.metadata-v1 feature missing %q boundary: %#v", want, feature)
				}
			}
		}
		if feature.ID == "ui.native-runtime" {
			if feature.Status != "current" || feature.Since != "v0.4.0" {
				t.Fatalf("ui.native-runtime lifecycle = status %q since %q, want current since v0.4.0", feature.Status, feature.Since)
			}
			for _, want := range []string{"production Linux-x64 native UI runtime path", "native runtime widget instances", "click/activate events", "state and widget updates", "tetra.ui.native-runtime.v1 smoke evidence", "metadata-only", "web-only", "native-shell sidecar-only", "macOS/Windows"} {
				if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
					t.Fatalf("ui.native-runtime feature missing %q boundary: %#v", want, feature)
				}
			}
		}
	}
	for _, status := range []string{"current", "planned", "post-v1"} {
		if !statusSeen[status] {
			t.Fatalf("features output missing %s status: %#v", status, report.Features)
		}
	}
	for id, wantStatus := range map[string]string{
		"cli.core":                                "current",
		"language.generics-mvp":                   "current",
		"language.protocol-conformance-mvp":       "current",
		"language.callable-mvp":                   "current",
		"targets.wasm-artifact-preflight":         "current",
		"stdlib.experimental-mirrors":             "current",
		"language.callable-level1":                "current",
		"language.enum-payload-match":             "current",
		"language.protocol-bound-generics-static": "current",
		"language.ownership-markers-mvp":          "current",
		"language.resource-lifetime-mvp":          "current",
		"actors.task-transfer-safety":             "current",
		"language.lifetime-ssa":                   "current",
		"language.callable-level2":                "current",
		"ui.metadata-v1":                          "current",
		"wasm.runtime-execution":                  "current",
		"eco.distributed-network":                 "post-v1",
		"actors.distributed-runtime":              "current",
		"ui.native-runtime":                       "current",
		"language.full-first-class-callables":     "current",
	} {
		if gotStatus := statusByID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
		}
	}
}

func TestFeaturesCommandRejectsUnsupportedFormat(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"features", "--format=yaml"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("features exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unsupported --format") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestFormatsCommandListsOfficialT4Family(t *testing.T) {
	var report struct {
		Formats []struct {
			Name      string `json:"name"`
			Extension string `json:"extension,omitempty"`
			FileName  string `json:"file_name,omitempty"`
			Role      string `json:"role"`
			Primary   bool   `json:"primary,omitempty"`
			Legacy    bool   `json:"legacy,omitempty"`
		} `json:"formats"`
	}
	runCLIJSONStdout(t, []string{"formats", "--format=json"}, 0, &report)
	seen := map[string]bool{}
	for _, format := range report.Formats {
		if format.Extension != "" {
			seen[format.Extension] = true
		}
		if format.FileName != "" {
			seen[format.FileName] = true
		}
	}
	for _, want := range []string{".t4", ".tetra", ".tdx", ".t4s", ".t4i", ".t4p", ".t4r", ".t4q", ".tneed", "Tetra.lock"} {
		if !seen[want] {
			t.Fatalf("formats output missing %s: %#v", want, report.Formats)
		}
	}
	byExtension := map[string]struct {
		Name    string
		Role    string
		Primary bool
		Legacy  bool
	}{}
	for _, format := range report.Formats {
		if format.Extension != "" {
			byExtension[format.Extension] = struct {
				Name    string
				Role    string
				Primary bool
				Legacy  bool
			}{Name: format.Name, Role: format.Role, Primary: format.Primary, Legacy: format.Legacy}
		}
	}
	if got := byExtension[".t4"]; got.Role != "source" || !got.Primary || got.Legacy {
		t.Fatalf(".t4 format metadata = %#v", got)
	}
	if got := byExtension[".tetra"]; got.Role != "source" || got.Primary || !got.Legacy {
		t.Fatalf(".tetra format metadata = %#v", got)
	}
}
