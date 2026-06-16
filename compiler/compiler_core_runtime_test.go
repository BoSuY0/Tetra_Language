package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildCoreSerializationSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_serialization_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreFilesystemSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_filesystem_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetworkingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_networking_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_net_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreJSONSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_json_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreHTTPSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_http_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresPreparedSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_prepared_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresResultSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_result_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreAsyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_async_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_sync_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreTimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_time_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCryptoSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_crypto_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildExtensionSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "extension_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildGenericSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "generic_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestCoreV015SemanticDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "bool from int",
			src:  "func main() -> Int:\n  let x: Bool = 1\n  return 0\n",
			want: "type mismatch: expected 'bool', got 'i32'",
		},
		{
			name: "int from bool",
			src:  "func main() -> Int:\n  let x: Int = true\n  return x\n",
			want: "type mismatch: expected 'i32', got 'bool'",
		},
		{
			name: "duplicate enum case",
			src:  "enum Color:\n  case red\n  case red\nfunc main() -> Int:\n  return 0\n",
			want: "duplicate enum case 'red'",
		},
		{
			name: "unknown enum case",
			src:  "enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = Color.blue\n  return 0\n",
			want: "unknown enum case 'blue'",
		},
		{
			name: "compare different enums",
			src:  "enum A:\n  case one\nenum B:\n  case one\nfunc main() -> Int:\n  let a: A = A.one\n  let b: B = B.one\n  if a == b:\n    return 1\n  return 0\n",
			want: "cannot compare 'A' and 'B'",
		},
		{
			name: "invalid match pattern",
			src:  "enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = Color.red\n  match c:\n  case 1:\n    return 1\n  return 0\n",
			want: "match pattern type mismatch",
		},
		{
			name: "multiple defaults",
			src:  "func main() -> Int:\n  match 1:\n  case _:\n    return 1\n  case _:\n    return 2\n  return 0\n",
			want: "match default must be last",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected build error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildFlowStructSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2:\n  x: Int\n  y: Int\n\nfunc sum(v: Vec2) -> Int:\n  return v.x + v.y\n\nfunc main() -> Int:\n  let v: Vec2 = Vec2(x: 40, y: 2)\n  return sum(v)\n"
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowIslandSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int\nuses alloc, islands, io, mem:\n  island(64) as isl:\n    var msg: []UInt8 = core.island_make_u8(isl, 2)\n    msg[0] = 79\n    msg[1] = 10\n    print(msg)\n  return 0\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "O\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowUnsafeCapMemSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int\nuses alloc, capability, mem:\n  var out: Int = 1\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let _: Int = core.store_i32(p, 42, mem)\n    out = core.load_i32(p, mem)\n  return out\n"
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildBudgetedUnsafeCallsPreserveIRStack(t *testing.T) {
	src := `func main() -> Int
uses alloc, budget, capability, mem
budget(16):
    var out: Int = 1
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        out = core.load_i32(p, mem)
    return out
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("BuildFile: %v", err)
	}
}

func TestBuildBudgetRuntimeGuardAllowsAndFailsDeterministically(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	okSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(4):
    return tick()
`
	stdout, exitCode := buildAndRun(t, okSrc)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}

	failSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(0):
    return tick()
`
	err := buildOnly(t, failSrc)
	if err == nil {
		t.Fatalf("expected compile-time budget context rejection")
	}
	if !strings.Contains(err.Error(), "budget context for call to 'tick' requires caller budget at least 1, got 0") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildBudgetFailureABIReturnAndThrowShapes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name     string
		src      string
		wantExit int
	}{
		{
			name: "non throwing multi slot return defaults to zero slots",
			src: `struct Pair:
    x: Int
    y: Int

func one() -> Int:
    return 7

func pair() -> Pair
uses budget
budget(0):
    return Pair(x: one(), y: 8)

func main() -> Int
uses budget
budget(16):
    let p: Pair = pair()
    return p.x + p.y
`,
			wantExit: 0,
		},
		{
			name: "throwing compact result returns thrown default payload",
			src: `enum BudgetTrap:
    case exhausted
    case other

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted:
        21
    case BudgetTrap.other:
        22
`,
			wantExit: 21,
		},
		{
			name: "throwing non compact result returns thrown zero payload",
			src: `enum BudgetTrap:
    case exhausted(Int)
    case other(Int)

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted(code):
        30 + code
    case BudgetTrap.other(otherCode):
        40 + otherCode
`,
			wantExit: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, exitCode := buildAndRun(t, tt.src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

func TestBuildPrivacyConsentRuntimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(33, token)

func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
    return core.secret_unseal_i32(value, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let secret: secret.i32 = seal(token)
    return reveal(token, secret)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 33 {
		t.Fatalf("exit code = %d, want 33", exitCode)
	}
}

func TestBuildPrivacySealUnsealStaticOnlyDeterministicIdentity(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func roundtrip(token: consent.token, value: Int) -> Int
uses privacy
privacy
consent(token):
    let sealed: secret.i32 = core.secret_seal_i32(value, token)
    return core.secret_unseal_i32(sealed, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let first: Int = roundtrip(token, 17)
    let second: Int = roundtrip(token, 17)
    let third: Int = roundtrip(token, 9)
    return (first - second) + third
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}
}
