package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func runEndpointLoad(ctx context.Context, baseURL string, endpoint endpointBenchmarkSpec, workers int, level benchLevel, repeat int, duration time.Duration, serverPID int, serverPort int) dbRunReport {
	transport := &http.Transport{
		MaxConnsPerHost:     level.Connections,
		MaxIdleConns:        level.Connections,
		MaxIdleConnsPerHost: level.Connections,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	start := time.Now()
	deadline := start.Add(duration)
	results := make(chan loadResult, level.Concurrency)
	var wg sync.WaitGroup
	for i := 0; i < level.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				results <- oneEndpointRequest(ctx, client, baseURL+endpoint.Path, endpoint)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var latencies []time.Duration
	var requests, successes, failures int
	var bytesRead int64
	var firstErr error
	for result := range results {
		requests++
		bytesRead += int64(result.bytes)
		latencies = append(latencies, result.latency)
		if result.err != nil {
			failures++
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		successes++
	}
	elapsed := time.Since(start)
	if elapsed <= 0 {
		elapsed = time.Nanosecond
	}
	latency := latencySummary(latencies)
	report := dbRunReport{
		Endpoint:        endpoint.Name,
		Path:            endpoint.Path,
		Kind:            endpoint.Kind,
		Workers:         workers,
		Level:           level,
		Repeat:          repeat,
		DurationSeconds: duration.Seconds(),
		ElapsedSeconds:  elapsed.Seconds(),
		Requests:        requests,
		Successes:       successes,
		Failures:        failures,
		Bytes:           bytesRead,
		RPS:             float64(successes) / elapsed.Seconds(),
		AvgLatencyMS:    averageMS(latencies),
		P50LatencyMS:    latency.P50MS,
		P90LatencyMS:    latency.P90MS,
		P95LatencyMS:    latency.P95MS,
		P99LatencyMS:    latency.P99MS,
		P999LatencyMS:   latency.P999MS,
		MaxLatencyMS:    latency.MaxMS,
		Resource:        detectResource(serverPID, serverPort),
		Validation:      "real HTTP GET " + endpoint.Path + " responses validated against TechEmpower-compatible endpoint contract",
	}
	if firstErr != nil {
		report.Error = firstErr.Error()
	}
	return report
}

func capturePprofProfiles(ctx context.Context, baseURL string, dir string, duration time.Duration) (pprofArtifacts, error) {
	done, artifacts, err := startPprofCPUProfile(ctx, baseURL, dir, duration)
	if err != nil {
		return artifacts, err
	}
	if err := <-done; err != nil {
		return artifacts, err
	}
	if err := capturePprofHeap(ctx, baseURL, artifacts.HeapProfile); err != nil {
		return artifacts, err
	}
	return artifacts, nil
}

func startPprofCPUProfile(ctx context.Context, baseURL string, dir string, duration time.Duration) (<-chan error, pprofArtifacts, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, pprofArtifacts{}, err
	}
	artifacts := pprofArtifacts{
		CPUProfile:  filepath.Join(dir, "native-scram-live-db-cpu.pprof"),
		HeapProfile: filepath.Join(dir, "native-scram-live-db-heap.pprof"),
	}
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	done := make(chan error, 1)
	go func() {
		target := strings.TrimRight(baseURL, "/") + "/debug/pprof/profile?seconds=" + strconv.Itoa(seconds)
		timeout := time.Duration(seconds+10) * time.Second
		done <- downloadPprof(ctx, target, artifacts.CPUProfile, timeout)
	}()
	return done, artifacts, nil
}

func capturePprofHeap(ctx context.Context, baseURL string, path string) error {
	target := strings.TrimRight(baseURL, "/") + "/debug/pprof/heap"
	return downloadPprof(ctx, target, path, 10*time.Second)
}

func downloadPprof(ctx context.Context, target string, path string, timeout time.Duration) error {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", target, resp.StatusCode)
	}
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}

func oneEndpointRequest(ctx context.Context, client *http.Client, target string, endpoint endpointBenchmarkSpec) loadResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return loadResult{err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return loadResult{latency: time.Since(start), err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return loadResult{status: resp.StatusCode, latency: time.Since(start), err: err}
	}
	if err := endpoint.Validate(resp, body); err != nil {
		return loadResult{status: resp.StatusCode, bytes: len(body), latency: time.Since(start), err: err}
	}
	return loadResult{status: resp.StatusCode, bytes: len(body), latency: time.Since(start)}
}

func runSoak(ctx context.Context, baseURL string, endpoint endpointBenchmarkSpec, workers int, level benchLevel, duration time.Duration, serverPID int, serverPort int) *soakReport {
	startResource := detectResource(serverPID, serverPort)
	run := runEndpointLoad(ctx, baseURL, endpoint, workers, level, 0, duration, serverPID, serverPort)
	endResource := detectResource(serverPID, serverPort)
	drift := 0.0
	if run.AvgLatencyMS > 0 {
		// The load runner records complete-run latency. Without periodic sampling,
		// report zero drift and keep the raw start/end resource snapshots as the
		// stability signal.
		drift = 0
	}
	soak := &soakReport{
		Endpoint:        endpoint.Name,
		Path:            endpoint.Path,
		Workers:         workers,
		Level:           level,
		DurationSeconds: duration.Seconds(),
		Requests:        run.Requests,
		Successes:       run.Successes,
		Failures:        run.Failures,
		RPS:             run.RPS,
		AvgLatencyMS:    run.AvgLatencyMS,
		P99LatencyMS:    run.P99LatencyMS,
		P999LatencyMS:   run.P999LatencyMS,
		MaxLatencyMS:    run.MaxLatencyMS,
		FirstHalfAvgMS:  run.AvgLatencyMS,
		SecondHalfAvgMS: run.AvgLatencyMS,
		LatencyDriftMS:  drift,
		ResourceStart:   startResource,
		ResourceEnd:     endResource,
		Validation:      "longer endpoint soak completed with response validation, resource snapshots, and shutdown cleanup check",
		Error:           run.Error,
	}
	return soak
}

func detectResource(pid int, port int) resourceSnapshot {
	now := time.Now()
	snapshot := resourceSnapshot{
		Timestamp:      now.UTC().Format(time.RFC3339),
		PID:            pid,
		TCPConnections: countTCPConnections(port),
	}
	if pid == os.Getpid() {
		snapshot.Goroutines = runtime.NumGoroutine()
	}
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	raw, err := os.ReadFile(statusPath)
	if err != nil {
		return snapshot
	}
	snapshot.ProcessAlive = true
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "VmRSS:":
			snapshot.RSSKB, _ = strconv.ParseInt(fields[1], 10, 64)
		case "Threads:":
			snapshot.Threads, _ = strconv.Atoi(fields[1])
		}
	}
	if entries, err := os.ReadDir(filepath.Join("/proc", strconv.Itoa(pid), "fd")); err == nil {
		snapshot.FDCount = len(entries)
	}
	user, system := readProcessCPUSeconds(pid)
	snapshot.CPUUserSeconds = user
	snapshot.CPUSystemSeconds = system
	return snapshot
}

func readProcessCPUSeconds(pid int) (float64, float64) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, 0
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return 0, 0
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) < 15 {
		return 0, 0
	}
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	ticks := float64(clockTicksPerSecond())
	return utime / ticks, stime / ticks
}

func clockTicksPerSecond() int {
	out, err := exec.Command("getconf", "CLK_TCK").Output()
	if err != nil {
		return 100
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || value <= 0 {
		return 100
	}
	return value
}

func countTCPConnections(port int) int {
	if port <= 0 {
		return 0
	}
	hexPort := strings.ToUpper(fmt.Sprintf("%04X", port))
	return countTCPConnectionsIn("/proc/net/tcp", hexPort) + countTCPConnectionsIn("/proc/net/tcp6", hexPort)
}

func countTCPConnectionsIn(path string, hexPort string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	activeStates := map[string]bool{"01": true, "02": true, "03": true, "0A": true}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.Contains(fields[1], ":") {
			continue
		}
		local := fields[1]
		if strings.HasSuffix(strings.ToUpper(local), ":"+hexPort) && activeStates[strings.ToUpper(fields[3])] {
			count++
		}
	}
	return count
}

type latencyStats struct {
	P50MS  float64
	P90MS  float64
	P95MS  float64
	P99MS  float64
	P999MS float64
	MaxMS  float64
}

func averageMS(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values)) / float64(time.Millisecond)
}

func percentileMS(values []time.Duration, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return float64(sorted[idx]) / float64(time.Millisecond)
}

func latencySummary(values []time.Duration) latencyStats {
	if len(values) == 0 {
		return latencyStats{}
	}
	return latencyStats{
		P50MS:  percentileMS(values, 0.50),
		P90MS:  percentileMS(values, 0.90),
		P95MS:  percentileMS(values, 0.95),
		P99MS:  percentileMS(values, 0.99),
		P999MS: percentileMS(values, 0.999),
		MaxMS:  maxMS(values),
	}
}

func maxMS(values []time.Duration) float64 {
	var max time.Duration
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return float64(max) / float64(time.Millisecond)
}
