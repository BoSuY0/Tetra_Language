package compiler_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/target"
)

const tetraBillionCountLimit int64 = 1000000000

const tetraBillionCountSource = `func main() -> Int:
    var i: Int = 0
    while i < 1000000000:
        i = i + 1
    if i == 1000000000:
        return 0
    return 1
`

func TestTetraCountToOneBillion(t *testing.T) {
	if os.Getenv("TETRA_RUN_BILLION_COUNT") != "1" {
		t.Skip("set TETRA_RUN_BILLION_COUNT=1 to run the one-billion-count timing test")
	}
	tgt, ok := target.Host()
	if !ok {
		t.Skip("host target unsupported")
	}

	outPath := buildBillionCountProgram(t, tgt)
	start := time.Now()
	stdout, exitCode := runBillionCountProgram(t, outPath, billionCountTimeout(t))
	elapsed := time.Since(start)

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	t.Logf("Tetra counted to %d in %s (%.0f counts/s)", tetraBillionCountLimit, elapsed, float64(tetraBillionCountLimit)/elapsed.Seconds())
}

func BenchmarkTetraCountToOneBillion(b *testing.B) {
	tgt, ok := target.Host()
	if !ok {
		b.Skip("host target unsupported")
	}

	outPath := buildBillionCountProgram(b, tgt)
	timeout := billionCountTimeout(b)
	var totalRuntime time.Duration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		stdout, exitCode := runBillionCountProgram(b, outPath, timeout)
		elapsed := time.Since(start)
		totalRuntime += elapsed

		if stdout != "" {
			b.Fatalf("stdout = %q, want empty", stdout)
		}
		if exitCode != 0 {
			b.Fatalf("exit code = %d, want 0", exitCode)
		}
	}
	b.StopTimer()

	if b.N > 0 && totalRuntime > 0 {
		b.ReportMetric(float64(totalRuntime.Nanoseconds())/float64(b.N)/1e6, "run_ms/op")
		b.ReportMetric(float64(tetraBillionCountLimit*int64(b.N))/totalRuntime.Seconds(), "counts/s")
	}
}

func buildBillionCountProgram(tb testing.TB, tgt target.Target) string {
	tb.Helper()

	tmp := tb.TempDir()
	srcPath := filepath.Join(tmp, "billion_count.tetra")
	outPath := filepath.Join(tmp, "billion_count"+tgt.ExeExt)
	if err := os.WriteFile(srcPath, []byte(tetraBillionCountSource), 0o644); err != nil {
		tb.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, compiler.BuildOptions{ReleaseOptimize: true}); err != nil {
		tb.Fatalf("build billion count program: %v", err)
	}
	return outPath
}

func runBillionCountProgram(tb testing.TB, path string, timeout time.Duration) (string, int) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		tb.Fatalf("billion count program exceeded %s; output=%q", timeout, out.String())
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode()
		}
		tb.Fatalf("run billion count program: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func billionCountTimeout(tb testing.TB) time.Duration {
	tb.Helper()

	timeout := 5 * time.Minute
	if raw := os.Getenv("TETRA_BILLION_COUNT_TIMEOUT"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			tb.Fatalf("parse TETRA_BILLION_COUNT_TIMEOUT=%q: %v", raw, err)
		}
		timeout = parsed
	}
	return timeout
}
