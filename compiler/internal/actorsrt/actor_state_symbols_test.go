package actorsrt

import (
	"bytes"
	"strings"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func TestBuiltinRuntimeExportsActorStateSymbols(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_load") {
				t.Fatalf("runtime missing __tetra_actor_state_load")
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_store") {
				t.Fatalf("runtime missing __tetra_actor_state_store")
			}
		})
	}
}

func TestLinuxRuntimeExportsFilesystemSymbol(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if !hasSymbol(obj.Symbols, "__tetra_fs_exists") {
		t.Fatalf("linux runtime missing __tetra_fs_exists")
	}
}

func TestLinuxRuntimeExportsNetSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeExportsDistributedActorSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name       string
		build      func([]string) (*tobj.Object, error)
		wantNoop   bool
		wantActive bool
	}{
		{name: "linux-x64", build: BuildLinuxX64, wantActive: true},
		{name: "macos-x64", build: BuildMacOSX64, wantNoop: true},
		{name: "windows-x64", build: BuildWindowsX64, wantNoop: true},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			body, ok := symbolBody(obj, "__tetra_actor_net_pump")
			if !ok {
				t.Fatalf("runtime missing __tetra_actor_net_pump")
			}
			isNoop := len(body) >= 3 && body[0] == 0x31 && body[1] == 0xC0 && body[2] == 0xC3
			if tt.wantNoop && !isNoop {
				t.Fatalf("%s __tetra_actor_net_pump must be a no-op on non-Linux targets, body prefix=% x", tt.name, bodyPrefix(body, 8))
			}
			if tt.wantActive && isNoop {
				t.Fatalf("%s __tetra_actor_net_pump must be active, got no-op body", tt.name)
			}
		})
	}
}

func TestLinuxDistributedRuntimeUsesWideStackSubFor128ByteFrames(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	badSignedImm8Sub := []byte{0x48, 0x83, 0xEC, 0x80}
	goodImm32Sub := []byte{0x48, 0x81, 0xEC, 0x80, 0x00, 0x00, 0x00}
	for _, name := range []string{"__tetra_actor_node_connect", "__tetra_actor_net_pump"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("runtime missing %s", name)
		}
		if bytes.Contains(body, badSignedImm8Sub) {
			t.Fatalf("%s uses signed imm8 stack subtraction for 128-byte frame", name)
		}
		if !bytes.Contains(body, goodImm32Sub) {
			t.Fatalf("%s missing imm32 stack subtraction for 128-byte frame, prefix=% x", name, bodyPrefix(body, 16))
		}
	}
}

func TestNonLinuxRuntimesDoNotExportDistributedActorSymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build([]string{"main", "worker"})
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			for _, name := range []string{
				"__tetra_actor_node_connect",
				"__tetra_actor_spawn_remote",
				"__tetra_actor_node_status",
			} {
				if hasSymbol(obj.Symbols, name) {
					t.Fatalf("%s runtime must not export Linux distributed actor symbol %s", tt.name, name)
				}
			}
		})
	}
}

func TestRuntimeBuildersRejectInvalidEntrySymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	cases := []struct {
		name    string
		entries []string
		want    string
	}{
		{
			name:    "missing_main",
			entries: nil,
			want:    "missing entry symbols",
		},
		{
			name:    "empty_main",
			entries: []string{""},
			want:    "missing entry symbols",
		},
		{
			name:    "empty_spawn_entry",
			entries: []string{"main", ""},
			want:    "empty runtime entry symbol at index 1",
		},
		{
			name:    "duplicate_entry",
			entries: []string{"main", "worker", "worker"},
			want:    "duplicate runtime entry symbol 'worker'",
		},
	}

	for _, builder := range builders {
		for _, tc := range cases {
			t.Run(builder.name+"/"+tc.name, func(t *testing.T) {
				_, err := builder.build(tc.entries)
				if err == nil {
					t.Fatalf("expected invalid entry symbol error")
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want substring %q", err, tc.want)
				}
			})
		}
	}
}

func hasSymbol(symbols []tobj.Symbol, want string) bool {
	for _, sym := range symbols {
		if sym.Name == want {
			return true
		}
	}
	return false
}

func symbolBody(obj *tobj.Object, want string) ([]byte, bool) {
	start := -1
	end := len(obj.Code)
	for _, sym := range obj.Symbols {
		offset := int(sym.Offset)
		if sym.Name == want {
			start = offset
			continue
		}
		if start >= 0 && offset > start && offset < end {
			end = offset
		}
	}
	if start < 0 || start > len(obj.Code) || end < start {
		return nil, false
	}
	return obj.Code[start:end], true
}

func bodyPrefix(body []byte, n int) []byte {
	if len(body) < n {
		return body
	}
	return body[:n]
}
