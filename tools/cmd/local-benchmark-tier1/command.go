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
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
	"time"
)

const rssSampleInterval = 500 * time.Microsecond

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

func runIterationsWithHeapTelemetry(timeout time.Duration, argv []string, env []string, iterations int, stdoutPath string, stderrPath string, telemetryDir string, benchmarkName string, outDir string) ([]float64, *runtimeHeapEvidence, *runtimeRSSEvidence, []string, error) {
	var stdoutAll bytes.Buffer
	var stderrAll bytes.Buffer
	var measurements []float64
	var firstErr error
	var selectedHeap *runtimeHeapEvidence
	var selectedRSS *runtimeRSSEvidence
	var artifacts []string
	var heapSamples []heapTelemetrySummarySample
	var rssSamples []rssTelemetrySummarySample
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		return nil, nil, nil, nil, err
	}
	sourceSidecar := filepath.Join(telemetryDir, filepath.Base(argv[0])+".heap.json")
	for i := 0; i < iterations; i++ {
		_ = os.Remove(sourceSidecar)
		stdout, stderr, exitCode, elapsed, rssSample, err := runCommandWithRSS(timeout, argv, env, benchmarkName, rssSampleInterval)
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

		heapArtifact := filepath.Join(outDir, "artifacts", "heap-telemetry", benchmarkName, fmt.Sprintf("iteration-%02d.heap.json", i+1))
		if copyErr := copyFile(sourceSidecar, heapArtifact); copyErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("runtime heap telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, copyErr)
			}
		} else {
			artifacts = append(artifacts, heapArtifact)
			sample, readErr := heaptelemetry.ReadFile(heapArtifact, outDir)
			if readErr != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("runtime heap telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, readErr)
				}
			} else {
				heapSamples = append(heapSamples, heapTelemetrySummarySample{
					Iteration:           i + 1,
					Artifact:            heapArtifact,
					HeapCurrentBytes:    sample.HeapCurrentBytes,
					HeapPeakBytes:       sample.HeapPeakBytes,
					HeapTotalAllocBytes: sample.HeapTotalAllocBytes,
					HeapAllocationCount: sample.HeapAllocationCount,
					BytesRequested:      sample.BytesRequested,
					BytesReserved:       sample.BytesReserved,
				})
				candidate := &runtimeHeapEvidence{SourceArtifact: heapArtifact, Sample: sample}
				if selectedHeap == nil || runtimeHeapSampleBetter(candidate.Sample, selectedHeap.Sample) {
					selectedHeap = candidate
				}
			}
		}

		rssArtifact := filepath.Join(outDir, "artifacts", "rss-telemetry", benchmarkName, fmt.Sprintf("iteration-%02d.rss.json", i+1))
		if writeErr := writeJSON(rssArtifact, rssSample); writeErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("runtime RSS telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, writeErr)
			}
			continue
		}
		artifacts = append(artifacts, rssArtifact)
		validRSSSample, readErr := rsstelemetry.ReadFile(rssArtifact, outDir)
		if readErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("runtime RSS telemetry sidecar for %s iteration %d: %w", benchmarkName, i+1, readErr)
			}
			continue
		}
		rssSamples = append(rssSamples, rssTelemetrySummarySample{
			Iteration:            i + 1,
			Artifact:             rssArtifact,
			SampleCount:          validRSSSample.SampleCount,
			RSSCurrentBytes:      validRSSSample.RSSCurrentBytes,
			RSSPeakBytes:         validRSSSample.RSSPeakBytes,
			RSSPeakSource:        validRSSSample.RSSPeakSource,
			RUMaxRSSRaw:          validRSSSample.RUMaxRSSRaw,
			RUMaxRSSUnit:         validRSSSample.RUMaxRSSUnit,
			SampleIntervalMicros: validRSSSample.SampleIntervalMicros,
		})
		rssCandidate := &runtimeRSSEvidence{SourceArtifact: rssArtifact, Sample: validRSSSample}
		if selectedRSS == nil || runtimeRSSSampleBetter(rssCandidate.Sample, selectedRSS.Sample) {
			selectedRSS = rssCandidate
		}
	}
	if len(heapSamples) > 0 {
		summaryPath, err := writeHeapTelemetrySummary(outDir, benchmarkName, selectedHeap, heapSamples)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
		} else {
			artifacts = append(artifacts, summaryPath)
		}
	}
	if len(rssSamples) > 0 {
		summaryPath, err := writeRSSTelemetrySummary(outDir, benchmarkName, selectedRSS, rssSamples)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
		} else {
			artifacts = append(artifacts, summaryPath)
		}
	}
	_ = os.WriteFile(stdoutPath, stdoutAll.Bytes(), 0o644)
	_ = os.WriteFile(stderrPath, stderrAll.Bytes(), 0o644)
	return measurements, selectedHeap, selectedRSS, artifacts, firstErr
}

func runtimeHeapSampleBetter(candidate heaptelemetry.Sample, current heaptelemetry.Sample) bool {
	if candidate.HeapPeakBytes != current.HeapPeakBytes {
		return candidate.HeapPeakBytes > current.HeapPeakBytes
	}
	if candidate.HeapTotalAllocBytes != current.HeapTotalAllocBytes {
		return candidate.HeapTotalAllocBytes > current.HeapTotalAllocBytes
	}
	return candidate.HeapAllocationCount > current.HeapAllocationCount
}

func runtimeRSSSampleBetter(candidate rsstelemetry.Sample, current rsstelemetry.Sample) bool {
	if candidate.RSSPeakBytes != current.RSSPeakBytes {
		return candidate.RSSPeakBytes > current.RSSPeakBytes
	}
	if candidate.SampleCount > 0 != (current.SampleCount > 0) {
		return candidate.SampleCount > 0
	}
	return candidate.SampleCount > current.SampleCount
}

func writeHeapTelemetrySummary(outDir string, benchmarkName string, selected *runtimeHeapEvidence, samples []heapTelemetrySummarySample) (string, error) {
	path := filepath.Join(outDir, "artifacts", "heap-telemetry", benchmarkName, "summary.json")
	selectedArtifact := ""
	if selected != nil {
		selectedArtifact = selected.SourceArtifact
	}
	data := map[string]any{
		"schema":            "tetra.local_benchmark.runtime_heap_summary.v1",
		"benchmark":         benchmarkName,
		"method":            heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"selected_artifact": selectedArtifact,
		"samples":           samples,
	}
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func writeRSSTelemetrySummary(outDir string, benchmarkName string, selected *runtimeRSSEvidence, samples []rssTelemetrySummarySample) (string, error) {
	path := filepath.Join(outDir, "artifacts", "rss-telemetry", benchmarkName, "summary.json")
	selectedArtifact := ""
	if selected != nil {
		selectedArtifact = selected.SourceArtifact
	}
	data := map[string]any{
		"schema":            "tetra.local_benchmark.process_rss_summary.v1",
		"benchmark":         benchmarkName,
		"method":            rsstelemetry.MethodLinuxProcfsWait4RSSSamplerV1,
		"selected_artifact": selectedArtifact,
		"samples":           samples,
	}
	if err := writeJSON(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func copyFile(src string, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
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

func runCommandWithRSS(timeout time.Duration, argv []string, env []string, program string, sampleInterval time.Duration) ([]byte, []byte, int, time.Duration, rsstelemetry.Sample, error) {
	if sampleInterval <= 0 {
		sampleInterval = rssSampleInterval
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	sample := rsstelemetry.Sample{
		Schema:               rsstelemetry.Schema,
		Method:               rsstelemetry.MethodLinuxProcfsWait4RSSSamplerV1,
		Program:              program,
		TargetOS:             runtime.GOOS,
		TargetArch:           runtime.GOARCH,
		StartedUnixNano:      start.UnixNano(),
		SampleIntervalMicros: uint64(sampleInterval / time.Microsecond),
	}
	if err := cmd.Start(); err != nil {
		elapsed := time.Since(start)
		sample.FinishedUnixNano = time.Now().UnixNano()
		sample.ExitStatus = 255
		return stdout.Bytes(), stderr.Bytes(), -1, elapsed, sample, err
	}
	if cmd.Process != nil {
		sample.PID = cmd.Process.Pid
	}

	recordRSS := func() {
		rssBytes, ok := readProcessRSSBytes(sample.PID)
		if !ok || rssBytes == 0 {
			return
		}
		sample.RSSCurrentBytes = rssBytes
		sample.SampleCount++
		if len(sample.Samples) < 64 {
			sample.Samples = append(sample.Samples, rsstelemetry.RSSSample{
				UnixNano: time.Now().UnixNano(),
				RSSBytes: rssBytes,
			})
		}
	}
	recordRSS()

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	var waitErr error
	done := false
	for !done {
		select {
		case waitErr = <-waitCh:
			done = true
		case <-ticker.C:
			recordRSS()
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			waitErr = <-waitCh
			done = true
		}
	}

	elapsed := time.Since(start)
	sample.FinishedUnixNano = time.Now().UnixNano()
	rawMaxRSS, peakBytes, peakUnit, ok := processStateMaxRSS(cmd.ProcessState)
	if ok {
		sample.RUMaxRSSRaw = rawMaxRSS
		sample.RUMaxRSSUnit = peakUnit
		sample.RSSPeakBytes = peakBytes
		sample.RSSPeakSource = rsstelemetry.PeakSourceWait4RusageMaxRSS
	}
	if sample.RSSPeakBytes < sample.RSSCurrentBytes {
		sample.Notes = append(sample.Notes, "wait4 peak RSS was below live current RSS; peak adjusted to current sample")
		sample.RSSPeakBytes = sample.RSSCurrentBytes
		if sample.RSSPeakSource == "" {
			sample.RSSPeakSource = "procfs_vmrss_current_fallback"
		}
	}

	exitCode := 0
	err := waitErr
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		err = ctx.Err()
		sample.ExitStatus = 124
	} else if waitErr == nil {
		sample.ExitStatus = 0
	} else if exit, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exit.ExitCode()
		if exitCode >= 0 {
			sample.ExitStatus = exitCode
		} else {
			sample.ExitStatus = 255
		}
	} else {
		exitCode = -1
		sample.ExitStatus = 255
	}
	return stdout.Bytes(), stderr.Bytes(), exitCode, elapsed, sample, err
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
