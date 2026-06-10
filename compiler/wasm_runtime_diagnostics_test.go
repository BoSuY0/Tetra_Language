package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestWasmRuntimeBuiltinsUseTargetAwareDiagnostics(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantMessage string
	}{
		{
			name: "actors",
			src: `func worker() -> Int
uses actors:
    return core.recv()

func main() -> Int
uses actors:
    let worker: actor = core.spawn("worker")
    return 0
`,
			wantMessage: "actors runtime not supported on %s",
		},
		{
			name: "distributed-actors",
			src: `func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`,
			wantMessage: "distributed actors runtime not supported on %s",
		},
		{
			name: "task",
			src: `func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			wantMessage: "task runtime not supported on %s",
		},
		{
			name: "time",
			src: `func main() -> Int
uses runtime:
    return core.time_now_ms()
`,
			wantMessage: "time runtime not supported on %s",
		},
		{
			name: "filesystem-classifier",
			src: `func main() -> Int:
    return 0
`,
			wantMessage: "filesystem runtime not supported on %s",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
				target := target
				t.Run(target, func(t *testing.T) {
					dir := t.TempDir()
					srcPath := filepath.Join(dir, "main.tetra")
					outPath := filepath.Join(dir, "app.wasm")
					if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
						t.Fatalf("write source: %v", err)
					}

					var err error
					if tc.name == "filesystem-classifier" {
						err = rejectUnsupportedWASMRuntimeBuiltins([]IRFunc{{
							Name: "main",
							Instrs: []ir.IRInstr{{
								Kind:     ir.IRCall,
								Name:     "__tetra_fs_exists",
								ArgSlots: 3,
								RetSlots: 1,
							}},
						}}, target)
					} else {
						_, err = BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
					}
					if err == nil {
						t.Fatalf("expected wasm runtime diagnostic")
					}
					diag := DiagnosticFromError(err)
					if diag.Code != "TETRA3003" || diag.Severity != "error" {
						t.Fatalf("diagnostic identity = %#v", diag)
					}
					want := strings.ReplaceAll(tc.wantMessage, "%s", target)
					if diag.Message != want {
						t.Fatalf("message = %q, want %q", diag.Message, want)
					}
					if strings.Contains(err.Error(), "unsupported IR instruction") || strings.Contains(err.Error(), "unsupported symbol") {
						t.Fatalf("diagnostic leaked generic backend text: %v", err)
					}
				})
			}
		})
	}
}
