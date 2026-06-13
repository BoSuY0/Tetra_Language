package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func runIterations(timeout time.Duration, argv []string, env []string, iterations int, stdoutPath string, stderrPath string) ([]float64, error) {
	var stdoutAll bytes.Buffer
	var stderrAll bytes.Buffer
	var measurements []float64
	var firstErr error
	for i := 0; i < iterations; i++ {
		stdout, stderr, exitCode, elapsed, err := runCommand(timeout, argv, env)
		fmt.Fprintf(&stdoutAll, "== iteration %d exit=%d elapsed_ms=%.3f ==\n", i+1, exitCode, millis(elapsed))
		stdoutAll.Write(stdout)
		if len(stdout) > 0 && stdout[len(stdout)-1] != '\n' {
			stdoutAll.WriteByte('\n')
		}
		fmt.Fprintf(&stderrAll, "== iteration %d exit=%d elapsed_ms=%.3f ==\n", i+1, exitCode, millis(elapsed))
		stderrAll.Write(stderr)
		if len(stderr) > 0 && stderr[len(stderr)-1] != '\n' {
			stderrAll.WriteByte('\n')
		}
		measurements = append(measurements, millis(elapsed))
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	_ = os.WriteFile(stdoutPath, stdoutAll.Bytes(), 0o644)
	_ = os.WriteFile(stderrPath, stderrAll.Bytes(), 0o644)
	return measurements, firstErr
}

func runCaptured(timeout time.Duration, argv []string, env []string, stdoutPath string, stderrPath string) (int, time.Duration, error) {
	stdout, stderr, exitCode, elapsed, err := runCommand(timeout, argv, env)
	_ = os.WriteFile(stdoutPath, stdout, 0o644)
	_ = os.WriteFile(stderrPath, stderr, 0o644)
	return exitCode, elapsed, err
}

func runCommand(timeout time.Duration, argv []string, env []string) ([]byte, []byte, int, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.Bytes(), stderr.Bytes(), -1, elapsed, ctx.Err()
	}
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), 0, elapsed, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return stdout.Bytes(), stderr.Bytes(), exit.ExitCode(), elapsed, err
	}
	return stdout.Bytes(), stderr.Bytes(), -1, elapsed, err
}

func commandOutput(timeout time.Duration, argv []string, env []string) string {
	stdout, stderr, _, _, err := runCommand(timeout, argv, env)
	text := strings.TrimSpace(string(stdout))
	if text == "" {
		text = strings.TrimSpace(string(stderr))
	}
	if err != nil && text == "" {
		return err.Error()
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return text
}

func compilerVersions(timeout time.Duration, env []string, tetraTool string) map[string]string {
	return map[string]string{
		"tetra": commandOutput(timeout, []string{tetraTool, "version"}, env),
		"c":     commandOutput(timeout, []string{"clang", "--version"}, env),
		"cpp":   commandOutput(timeout, []string{"clang++", "--version"}, env),
		"rust":  commandOutput(timeout, []string{"rustc", "--version", "--verbose"}, env),
	}
}

func gitCommit(timeout time.Duration, env []string) string {
	out := commandOutput(timeout, []string{"git", "rev-parse", "HEAD"}, env)
	if strings.TrimSpace(out) == "" {
		return "unknown"
	}
	return out
}

func detectTargetCPU() string {
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						if cpu := strings.TrimSpace(parts[1]); cpu != "" {
							return cpu
						}
					}
				}
			}
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

func commandEnv(root string) []string {
	env := os.Environ()
	env = append(env, "GOCACHE="+filepath.Join(root, ".cache", "go-build-p25-tier1"))
	return env
}

func writeOptimizerArtifact(outDir string) (string, error) {
	path := filepath.Join(outDir, "artifacts", "optimizer-validation.json")
	data := map[string]any{
		"schema": "tetra.local_benchmark.optimizer_validation_metadata.v1",
		"status": "current_supported_subset",
		"artifacts": []string{
			"compiler/translation_validation_v2.go",
			"compiler/internal/opt/manager.go",
			"compiler/internal/validation/validation.go",
		},
		"non_claim": "optimizer validation metadata is current supported-subset evidence, not exhaustive optimizer completeness",
	}
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}
