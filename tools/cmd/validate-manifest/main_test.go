package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateManifestAcceptsGeneratedShape(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "formats": [
    {"name":"T4 Source Format","extension":".t4","role":"source","description":"Tetra source file","primary":true},
    {"name":"Legacy Tetra Source Format","extension":".tetra","role":"source","description":"Legacy Tetra source file","legacy":true},
    {"name":"Todex Fragment","extension":".tdx","role":"todex-fragment","description":"Todex encrypted semantic fragment"},
    {"name":"T4 Seed","extension":".t4s","role":"offline-seed","description":"Tetra Seed offline bundle"},
    {"name":"T4 Interface","extension":".t4i","role":"interface","description":"T4 interface file"},
    {"name":"T4 Proof","extension":".t4p","role":"proof","description":"T4 proof file"},
    {"name":"T4 Replay","extension":".t4r","role":"replay","description":"T4 replay file"},
    {"name":"T4 Quest","extension":".t4q","role":"quest","description":"T4 executable quest file"},
    {"name":"Tetra NeedMap","extension":".tneed","role":"needmap","description":"NeedMap file"},
    {"name":"Tetra Semantic Lock","file_name":"Tetra.lock","role":"semantic-lock","description":"Tetra semantic lockfile"}
  ],
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","collect_imports":false},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","collect_imports":true},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","collect_imports":false}
  ],
  "builtins": [
    {"name":"core.load_i32","param_types":["ptr","cap.mem"],"return_type":"i32","effects":["mem"],"unsafe_policy":"always"},
    {"name":"core.print","aliases":["print"],"param_types":["str"],"return_type":"i32","effects":["io"],"unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateManifestRejectsNullTargets(t *testing.T) {
	manifest := `{"compiler_version":"v0.6.0","targets":null,"builtins":[],"runtime_abi":{}}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnknownFields(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","collect_imports":false},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","collect_imports":true},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","collect_imports":false}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never","extra":true}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsDuplicateBuiltin(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate builtin core.print") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": [],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingTimeRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "time_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsInvalidUnsafePolicy(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"sometimes"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid unsafe_policy") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialTargetSurface(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialRuntimeABI(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnsortedTargets(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must follow supported target order") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnsortedBuiltins(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"}
  ],
  "builtins": [
    {"name":"core.z","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.a","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "builtins must be sorted") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runManifestValidator(t *testing.T, manifest string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--manifest", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
