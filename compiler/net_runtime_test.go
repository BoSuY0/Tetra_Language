package compiler

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

func TestNetRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredNetRuntimeSymbols()
	want := []string{
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
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("networking runtime symbols = %#v, want %#v", got, want)
	}
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_net_socket_tcp4", params: 1, rets: 1},
		{name: "__tetra_net_bind_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_connect_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_listen", params: 3, rets: 1},
		{name: "__tetra_net_accept4", params: 3, rets: 1},
		{name: "__tetra_net_read", params: 6, rets: 1},
		{name: "__tetra_net_recv", params: 6, rets: 1},
		{name: "__tetra_net_write", params: 6, rets: 1},
		{name: "__tetra_net_send", params: 6, rets: 1},
		{name: "__tetra_net_epoll_create", params: 1, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_delete", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one_into", params: 5, rets: 1},
		{name: "__tetra_net_set_nonblocking", params: 2, rets: 1},
		{name: "__tetra_net_set_reuseport", params: 2, rets: 1},
		{name: "__tetra_net_set_tcp_nodelay", params: 2, rets: 1},
		{name: "__tetra_net_close", params: 2, rets: 1},
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

func TestLinuxX86BasicNetRuntimeObjectExportsSocketNonblockingClose(t *testing.T) {
	rt := buildLinuxX86BasicNetRuntimeObject()
	if rt.Target != "linux-x86" {
		t.Fatalf("runtime target = %q, want linux-x86", rt.Target)
	}
	if rt.Module != "__linux_x86_netrt" {
		t.Fatalf("runtime module = %q, want __linux_x86_netrt", rt.Module)
	}
	wantSymbols := []string{
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
	}
	if !runtimeSymbolsMatch(rt.Symbols, wantSymbols) {
		t.Fatalf("runtime symbols = %#v, want %v in offset order", rt.Symbols, wantSymbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf("runtime object must be self-contained, data=%d relocs=%#v", len(rt.Data), rt.Relocs)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateRuntimeObjectSymbols(rt, "missing networking runtime object", wantSymbols); err != nil {
		t.Fatalf("validate x86 basic net runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"socketcall syscall": {0xB8, 0x66, 0x00, 0x00, 0x00},
		"socket operation":   {0xBB, 0x01, 0x00, 0x00, 0x00},
		"bind operation":     {0xBB, 0x02, 0x00, 0x00, 0x00},
		"connect operation":  {0xBB, 0x03, 0x00, 0x00, 0x00},
		"listen operation":   {0xBB, 0x04, 0x00, 0x00, 0x00},
		"send operation":     {0xBB, 0x09, 0x00, 0x00, 0x00},
		"recv operation":     {0xBB, 0x0A, 0x00, 0x00, 0x00},
		"setsockopt op":      {0xBB, 0x0E, 0x00, 0x00, 0x00},
		"accept4 operation":  {0xBB, 0x12, 0x00, 0x00, 0x00},
		"read syscall":       {0xB8, 0x03, 0x00, 0x00, 0x00},
		"write syscall":      {0xB8, 0x04, 0x00, 0x00, 0x00},
		"epoll_create1":      {0xB8, 0x49, 0x01, 0x00, 0x00},
		"epoll_ctl":          {0xB8, 0xFF, 0x00, 0x00, 0x00},
		"epoll_wait":         {0xB8, 0x00, 0x01, 0x00, 0x00},
		"fcntl syscall":      {0xB8, 0x37, 0x00, 0x00, 0x00},
		"nonblocking flag":   {0x0D, 0x00, 0x08, 0x00, 0x00},
		"close syscall":      {0xB8, 0x06, 0x00, 0x00, 0x00},
		"int80 syscall":      {0xCD, 0x80},
		"preserved return":   {0x5B, 0x5D, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
}

func TestLinuxX32BasicNetRuntimeObjectExportsSocketNonblockingClose(t *testing.T) {
	rt := buildLinuxX32BasicNetRuntimeObject()
	if rt.Target != "linux-x32" {
		t.Fatalf("runtime target = %q, want linux-x32", rt.Target)
	}
	if rt.Module != "__linux_x32_netrt" {
		t.Fatalf("runtime module = %q, want __linux_x32_netrt", rt.Module)
	}
	wantSymbols := []string{
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
	}
	if !runtimeSymbolsMatch(rt.Symbols, wantSymbols) {
		t.Fatalf("runtime symbols = %#v, want %v in offset order", rt.Symbols, wantSymbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf("runtime object must be self-contained, data=%d relocs=%#v", len(rt.Data), rt.Relocs)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateRuntimeObjectSymbols(rt, "missing networking runtime object", wantSymbols); err != nil {
		t.Fatalf("validate x32 basic net runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"x32 socket syscall":  {0xB8, 0x29, 0x00, 0x00, 0x40},
		"x32 bind syscall":    {0xB8, 0x31, 0x00, 0x00, 0x40},
		"x32 connect syscall": {0xB8, 0x2A, 0x00, 0x00, 0x40},
		"x32 listen syscall":  {0xB8, 0x32, 0x00, 0x00, 0x40},
		"x32 accept4 syscall": {0xB8, 0x20, 0x01, 0x00, 0x40},
		"x32 read syscall":    {0xB8, 0x00, 0x00, 0x00, 0x40},
		"x32 write syscall":   {0xB8, 0x01, 0x00, 0x00, 0x40},
		"x32 send syscall":    {0xB8, 0x2C, 0x00, 0x00, 0x40},
		"x32 recv syscall":    {0xB8, 0x05, 0x02, 0x00, 0x40},
		"x32 setsockopt":      {0xB8, 0x1D, 0x02, 0x00, 0x40},
		"x32 epoll_wait":      {0xB8, 0xE8, 0x00, 0x00, 0x40},
		"x32 epoll_ctl":       {0xB8, 0xE9, 0x00, 0x00, 0x40},
		"x32 epoll_create1":   {0xB8, 0x23, 0x01, 0x00, 0x40},
		"x32 fcntl syscall":   {0xB8, 0x48, 0x00, 0x00, 0x40},
		"nonblocking flag":    {0x0D, 0x00, 0x08, 0x00, 0x00},
		"x32 close syscall":   {0xB8, 0x03, 0x00, 0x00, 0x40},
		"syscall instruction": {0x0F, 0x05},
		"return":              {0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
	if bytes.Contains(rt.Code, []byte{0xB8, 0x03, 0x00, 0x00, 0x00}) {
		t.Fatalf("x32 net close runtime emitted plain x64 close syscall: % x", rt.Code)
	}
}

func runtimeSymbolsMatch(symbols []Symbol, names []string) bool {
	if len(symbols) != len(names) {
		return false
	}
	var last uint32
	for i, name := range names {
		if symbols[i].Name != name {
			return false
		}
		if i > 0 && symbols[i].Offset <= last {
			return false
		}
		last = symbols[i].Offset
	}
	return true
}

func TestCollectNetRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe(cap: cap.io) -> Int
uses io:
    let fd: Int = core.net_socket_tcp4(cap)
    return core.net_close(fd, cap)

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
	if !collectNetRuntimeUsage(checked) {
		t.Fatalf("networking runtime usage was not collected")
	}
}

func TestValidateNetRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithNetRuntimeSignatures()
	if err := validateNetRuntimeObject(obj); err != nil {
		t.Fatalf("validate networking runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_net_set_nonblocking", 1, 1)
	err := validateNetRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected networking runtime signature mismatch")
	}
	if !strings.Contains(err.Error(), "runtime object symbol '__tetra_net_set_nonblocking' signature mismatch") ||
		!strings.Contains(err.Error(), "params=1 want=2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingNetSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if tgt.Triple != "linux-x64" {
		t.Skipf("networking runtime is linux-x64 only, host is %s", tgt.Triple)
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_net.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_net",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        return core.net_close(fd, cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "net_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected missing networking runtime symbol failure")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_net_socket_tcp4'") {
		t.Fatalf("unexpected error: %v", err)
	}
}
