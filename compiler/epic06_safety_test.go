package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEpic06EffectsCapabilitiesUnsafeOwnershipIslandPrivacyBudgetMatrix(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "Effect propagation positive through wrapper",
			src: `
func write() -> Int
uses io:
    print("ok\n")
    return 1

func main() -> Int
uses io:
    return write()
`,
		},
		{
			name: "Effect propagation negative missing caller uses",
			src: `
func write() -> Int
uses io:
    print("blocked\n")
    return 1

func main() -> Int:
    return write()
`,
			wantErr: "uses effect 'io'",
		},
		{
			name: "Capability positive with explicit capsule attenuation",
			src: `
func main() -> Int
uses capsule.mem, effects.cap.mem, effects.memory:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 5, mem)
        return core.load_i32(p, mem)
    return 0
`,
		},
		{
			name: "Capability negative missing capsule permission",
			src: `
func main() -> Int
uses effects.cap.mem, effects.memory:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        return core.load_i32(p, mem)
    return 0
`,
			wantErr: "capsule permission 'capsule.mem'",
		},
		{
			name: "Unsafe positive raw allocation in unsafe block",
			src: `
func main() -> Int
uses alloc, mem:
    unsafe:
        let p: ptr = core.alloc_bytes(4)
    return 0
`,
		},
		{
			name: "Unsafe negative raw allocation in safe code",
			src: `
func main() -> Int
uses alloc, mem:
    let p: ptr = core.alloc_bytes(4)
    return 0
`,
			wantErr: "only allowed in unsafe blocks",
		},
		{
			name: "Ownership positive distinct borrow and inout locals",
			src: `
func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    var b: Int = 2
    return mix(a, b)
`,
		},
		{
			name: "Ownership negative consumed value reused",
			src: `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`,
			wantErr: "cannot use consumed value 'a'",
		},
		{
			name: "Island region positive helper returns scoped slice",
			src: `
func make_buf(isl: island, n: Int) -> []u8
uses alloc, islands, mem:
    return core.island_make_u8(isl, n)

func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var buf: []u8 = make_buf(isl, 1)
        buf[0] = 7
    return 0
`,
		},
		{
			name: "Island region negative scoped slice escapes",
			src: `
func main() -> Int
uses alloc, islands, mem:
    var out: []u8 = make_u8(1)
    island(64) as isl:
        out = core.island_make_u8(isl, 1)
    return 0
`,
			wantErr: "escape",
		},
		{
			name: "Budget clause negative missing budget contract",
			src: `
func audit() -> Int
uses budget:
    return 1

func main() -> Int
uses budget:
    return audit()
`,
			wantErr: "uses effect 'budget' requires semantic clause 'budget'",
		},
		{
			name: "Budget clause negative policy group missing budget contract",
			src: `
func audit(token: consent.token) -> secret.i32
uses effects.policy
privacy
consent(token):
    return core.secret_seal_i32(1, token)

func main() -> Int:
    return 0
`,
			wantErr: "uses effect 'budget' requires semantic clause 'budget'",
		},
		{
			name: "Privacy and Budget positive via policy group",
			src: `
func audit(token: consent.token) -> secret.i32
uses effects.policy
privacy
consent(token)
budget(8):
    return core.secret_seal_i32(1, token)

func main() -> Int:
    return 0
`,
		},
		{
			name: "Privacy negative missing privacy clause",
			src: `
func main() -> Int
uses privacy:
    return 0
`,
			wantErr: "requires semantic clause 'privacy'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == "" {
				requireCheckOK(t, tt.src)
				return
			}
			requireCheckErrorContains(t, tt.src, tt.wantErr)
		})
	}
}

func TestEpic06OwnershipTransferForActorsAndTasks(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "Task ownership transfer positive",
			src: `
func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return take_task(task)
`,
		},
		{
			name: "Task ownership transfer negative reuse",
			src: `
func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = take_task(task)
    return value + core.task_join_i32(task)
`,
			wantErr: "cannot use consumed value 'task'",
		},
		{
			name: "Actor ownership transfer positive",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return take_actor(peer)
`,
		},
		{
			name: "Actor ownership transfer negative reuse",
			src: `
func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = take_actor(peer)
    return core.send(peer, 1)
`,
			wantErr: "cannot use consumed value 'peer'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr == "" {
				requireCheckFileOK(t, tt.src)
				return
			}
			requireCheckFileErrorContains(t, tt.src, tt.wantErr)
		})
	}
}

func TestEpic06OwnershipAliasRejectionMatrix(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "consume requires local value",
			src: `
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    return take(1 + 2)
`,
			want: "consume argument for 'take' must be a local value",
		},
		{
			name: "double consume same local in one call",
			src: `
func take_pair(left: consume Int, right: consume Int) -> Int:
    return left + right

func main() -> Int:
    let value: Int = 1
    return take_pair(value, value)
`,
			want: "value 'value' consumed more than once in call to 'take_pair'",
		},
		{
			name: "borrow and inout alias rejected",
			src: `
func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var value: Int = 1
    return mix(value, value)
`,
			want: "inout argument 'value' aliases borrowed argument in call to 'mix'",
		},
		{
			name: "inout and borrow alias rejected",
			src: `
func mix(write: inout Int, read: borrow Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var value: Int = 1
    return mix(value, value)
`,
			want: "borrowed argument 'value' aliases inout argument in call to 'mix'",
		},
		{
			name: "consume and inout alias rejected",
			src: `
func move_and_write(moved: consume Int, write: inout Int) -> Int:
    write = write + moved
    return write

func main() -> Int:
    var value: Int = 1
    return move_and_write(value, value)
`,
			want: "inout argument 'value' aliases consumed argument in call to 'move_and_write'",
		},
		{
			name: "inout and consume alias rejected",
			src: `
func write_and_move(write: inout Int, moved: consume Int) -> Int:
    write = write + moved
    return write

func main() -> Int:
    var value: Int = 1
    return write_and_move(value, value)
`,
			want: "consumed argument 'value' aliases inout argument in call to 'write_and_move'",
		},
		{
			name: "borrowed island slice cannot pass to non-borrow parameter",
			src: `
func use_buf(buf: []u8) -> Int:
    return 0

func forward(buf: borrow []u8) -> Int:
    return use_buf(buf)

func main() -> Int:
    return 0
`,
			want: "borrowed value derived from 'buf' cannot be passed to non-borrow parameter 1 of 'use_buf'",
		},
		{
			name: "borrowed island slice cannot pass as inout",
			src: `
func mutate(buf: inout []u8) -> Int:
    return 0

func forward(buf: borrow []u8) -> Int:
    return mutate(buf)

func main() -> Int:
    return 0
`,
			want: "borrowed value derived from 'buf' cannot be passed as inout to 'mutate'",
		},
		{
			name: "borrowed island slice cannot escape via return",
			src: `
func forward(buf: borrow []u8) -> []u8:
    return buf

func main() -> Int:
    return 0
`,
			want: "borrowed local 'buf' cannot escape via return",
		},
		{
			name: "borrowed island slice cannot escape through inout assignment",
			src: `
func forward(buf: borrow []u8, out: inout []u8) -> Int:
    out = buf
    return 0

func main() -> Int:
    return 0
`,
			want: "borrowed local 'buf' cannot escape via inout assignment to 'out'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckFileErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestEpic06CapabilityAndIslandExamplesUseAuditedEffects(t *testing.T) {
	for _, path := range []string{
		"examples/cap_mem_smoke.tetra",
		"examples/cap_mem_ptr_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
	} {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			src := readRepoFileForEpic06(t, path)
			requireCheckOK(t, src)
			if strings.Contains(path, "cap_") || strings.Contains(path, "mmio") {
				for _, want := range []string{"uses", "capability", "mem", "unsafe"} {
					if !strings.Contains(src, want) {
						t.Fatalf("%s missing audited capability marker %q", path, want)
					}
				}
			}
			if strings.Contains(path, "islands_") {
				for _, want := range []string{"uses", "islands", "mem"} {
					if !strings.Contains(src, want) {
						t.Fatalf("%s missing audited island marker %q", path, want)
					}
				}
			}
		})
	}
	t.Run("islands_double_free.tetra", func(t *testing.T) {
		src := readRepoFileForEpic06(t, "examples/islands_double_free.tetra")
		requireCheckErrorContains(t, src, "cannot use freed resource 'other'")
		for _, want := range []string{"uses", "islands", "mem"} {
			if !strings.Contains(src, want) {
				t.Fatalf("examples/islands_double_free.tetra missing audited island marker %q", want)
			}
		}
	})
}

func TestEpic06DocsAndReleaseGateAlignWithUsesCapabilityUnsafeOwnershipIslandCoverage(t *testing.T) {
	docs := map[string][]string{
		"docs/spec/effects_capabilities_privacy_v1.md": {
			"Function calls propagate callee effects transitively",
			"Privacy And Consent",
			"Budget",
			"Epic 06 release evidence",
		},
		"docs/spec/capabilities.md": {
			"Capabilities are not constructible in safe code",
			"capsule.mem",
			"Epic 06 coverage",
		},
		"docs/spec/unsafe.md": {
			"Unsafe-Only Builtins Registry",
			"Relationship to `uses`",
			"Epic 06 coverage",
		},
		"docs/spec/ownership_v1.md": {
			"consume T",
			"Actor And Task Transfer",
			"Epic 06 coverage",
		},
		"docs/spec/islands.md": {
			"Region Typing",
			"Scoped islands remain safe",
			"Epic 06 coverage",
		},
		"docs/user/ownership_effects_guide.md": {
			"Allowed patterns",
			"Forbidden patterns",
			"go test ./compiler/...",
		},
		"docs/checklists/v0_2_0_release_gate.md": {
			"Epic 06 safety gate",
			"Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget",
		},
	}

	for path, wants := range docs {
		path, wants := path, wants
		t.Run(filepath.Base(path), func(t *testing.T) {
			src := readRepoFileForEpic06(t, path)
			for _, want := range wants {
				if !strings.Contains(src, want) {
					t.Fatalf("%s missing %q", path, want)
				}
			}
		})
	}
}

func readRepoFileForEpic06(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
