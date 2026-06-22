package rambaseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"tetra_language/tools/internal/rsstelemetry"
)

const Schema = "tetra.ram.p0-baseline.v1"

const (
	defaultIterations = 5
	defaultWorkBytes  = 4 * 1024 * 1024
	allocatorMode     = "process_bump_small_heap_v0"
)

type Options struct {
	OutDir         string
	Iterations     int
	WorkBytes      int
	GitHead        string
	GitStatusShort string
	Command        []string
	Now            func() time.Time
}

type Result struct {
	OutDir              string
	ManifestPath        string
	RSSPath             string
	ValidatorOutputPath string
}

type baselineManifest struct {
	Schema                 string   `json:"schema"`
	GeneratedAt            string   `json:"generated_at"`
	GitHead                string   `json:"git_head"`
	GitDirty               bool     `json:"git_dirty"`
	GitStatusShort         []string `json:"git_status_short,omitempty"`
	TargetOS               string   `json:"target_os"`
	TargetArch             string   `json:"target_arch"`
	AllocatorMode          string   `json:"allocator_mode"`
	WorkloadKind           string   `json:"workload_kind"`
	Iterations             int      `json:"iterations"`
	WorkBytes              int      `json:"work_bytes"`
	TelemetrySchema        string   `json:"telemetry_schema"`
	TelemetryMethod        string   `json:"telemetry_method"`
	RSSSidecar             string   `json:"rss_sidecar"`
	HostFingerprint        string   `json:"host_fingerprint"`
	CommandManifest        string   `json:"command_manifest"`
	ValidatorOutput        string   `json:"validator_output"`
	SmapsRollupArtifacts   []string `json:"smaps_rollup_artifacts,omitempty"`
	ArtifactHashManifest   string   `json:"artifact_hash_manifest,omitempty"`
	ArtifactHashValidation string   `json:"artifact_hash_validation,omitempty"`
	Notes                  []string `json:"notes,omitempty"`
}

type hostFingerprint struct {
	Schema        string `json:"schema"`
	Hostname      string `json:"hostname"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	NumCPU        int    `json:"num_cpu"`
	KernelRelease string `json:"kernel_release,omitempty"`
	OSRelease     string `json:"os_release,omitempty"`
	CPUModel      string `json:"cpu_model,omitempty"`
}

type commandManifest struct {
	Schema   string          `json:"schema"`
	Commands []commandRecord `json:"commands"`
}

type commandRecord struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
}

func Run(opts Options) (Result, error) {
	if strings.TrimSpace(opts.OutDir) == "" {
		return Result{}, fmt.Errorf("baseline out dir is required")
	}
	if runtime.GOOS != rsstelemetry.TargetOSLinux {
		return Result{}, fmt.Errorf("ram P0 baseline harness requires linux procfs")
	}
	if opts.Iterations <= 0 {
		opts.Iterations = defaultIterations
	}
	if opts.WorkBytes <= 0 {
		opts.WorkBytes = defaultWorkBytes
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return Result{}, err
	}
	smapsDir := filepath.Join(opts.OutDir, "smaps")
	if err := os.MkdirAll(smapsDir, 0o755); err != nil {
		return Result{}, err
	}

	clock := monotonicClock{now: now}
	pid := os.Getpid()
	generatedAt := now().UTC().Format(time.RFC3339)
	sample := rsstelemetry.Sample{
		Schema:               rsstelemetry.SchemaV2,
		Method:               rsstelemetry.MethodLinuxProcfsPhaseRSSSamplerV2,
		Program:              "ram-p0-baseline",
		PID:                  pid,
		TargetOS:             runtime.GOOS,
		TargetArch:           runtime.GOARCH,
		StartedUnixNano:      clock.next(),
		ExitStatus:           0,
		WorkloadKind:         "steady_state",
		SampleIntervalMicros: 0,
		Notes: []string{
			"baseline harness measures its own controlled fixed-live-set workload",
			"no allocator reuse, release, or target parity improvement is claimed",
		},
	}

	var smapsArtifacts []string
	capture := func(phase string) error {
		rssBytes, ok := rsstelemetry.ReadProcessRSSBytes(pid)
		if !ok || rssBytes == 0 {
			return fmt.Errorf("read procfs RSS for phase %s", phase)
		}
		mappingCount, ok := rsstelemetry.ReadProcessMappingCount(pid)
		if !ok || mappingCount == 0 {
			return fmt.Errorf("read procfs mapping count for phase %s", phase)
		}
		if raw, ok := rsstelemetry.ReadProcessSmapsRollup(pid); ok {
			rel := filepath.ToSlash(filepath.Join("smaps", phase+".smaps_rollup"))
			if err := os.WriteFile(filepath.Join(opts.OutDir, filepath.FromSlash(rel)), raw, 0o644); err != nil {
				return err
			}
			smapsArtifacts = append(smapsArtifacts, rel)
		} else {
			sample.Notes = append(sample.Notes, "smaps_rollup unavailable for phase "+phase)
		}
		sample.RSSCurrentBytes = rssBytes
		sample.MappingCount = &mappingCount
		sample.SampleCount++
		if sample.RSSPeakBytes < rssBytes {
			sample.RSSPeakBytes = rssBytes
			sample.RSSPeakSource = "procfs_phase_samples_max_v2"
		}
		sample.Samples = append(sample.Samples, rsstelemetry.RSSSample{
			Phase:        phase,
			UnixNano:     clock.next(),
			RSSBytes:     rssBytes,
			MappingCount: &mappingCount,
		})
		return nil
	}

	if err := capture("startup"); err != nil {
		return Result{}, err
	}
	live := allocateAndTouch(opts.WorkBytes)
	if err := capture("post_warmup"); err != nil {
		return Result{}, err
	}
	for i := 1; i <= opts.Iterations; i++ {
		scratch := allocateAndTouch(opts.WorkBytes)
		if len(live) > 0 && len(live[0]) > 0 {
			live[i%len(live)][0]++
		}
		for j := range scratch {
			scratch[j] = nil
		}
		scratch = nil
		runtime.GC()
		if err := capture(fmt.Sprintf("steady_round_%d", i)); err != nil {
			return Result{}, err
		}
	}
	for i := range live {
		live[i] = nil
	}
	live = nil
	runtime.GC()
	debug.FreeOSMemory()
	if err := capture("post_drain"); err != nil {
		return Result{}, err
	}
	if err := capture("pre_exit"); err != nil {
		return Result{}, err
	}
	sample.FinishedUnixNano = clock.next()

	rssPath := filepath.Join(opts.OutDir, "rss-telemetry-v2.json")
	if err := writeJSON(rssPath, sample); err != nil {
		return Result{}, err
	}
	validatorOutputPath := filepath.Join(opts.OutDir, "validator-output.txt")
	validatorOutput := "validator: rsstelemetry.ReadFile\nschema: " +
		rsstelemetry.SchemaV2 + "\nresult: pass\n"
	if _, err := rsstelemetry.ReadFile(rssPath, opts.OutDir); err != nil {
		validatorOutput = "validator: rsstelemetry.ReadFile\nschema: " +
			rsstelemetry.SchemaV2 + "\nresult: fail\nerror: " + err.Error() + "\n"
		_ = os.WriteFile(validatorOutputPath, []byte(validatorOutput), 0o644)
		return Result{}, err
	}
	if err := os.WriteFile(validatorOutputPath, []byte(validatorOutput), 0o644); err != nil {
		return Result{}, err
	}

	hostPath := filepath.Join(opts.OutDir, "host-fingerprint.json")
	if err := writeJSON(hostPath, readHostFingerprint()); err != nil {
		return Result{}, err
	}
	commandPath := filepath.Join(opts.OutDir, "command-manifest.json")
	if err := writeJSON(commandPath, commandManifest{
		Schema: "tetra.ram.p0-baseline.commands.v1",
		Commands: []commandRecord{
			{Name: "ram-p0-baseline", Command: append([]string(nil), opts.Command...)},
			{
				Name: "artifact-hashes-write",
				Command: []string{
					"go", "run", "./tools/cmd/validate-artifact-hashes",
					"--write",
					"--root", opts.OutDir,
					"--out", filepath.Join(opts.OutDir, "artifact-hashes.json"),
				},
			},
			{
				Name: "artifact-hashes-validate",
				Command: []string{
					"go", "run", "./tools/cmd/validate-artifact-hashes",
					"--manifest", filepath.Join(opts.OutDir, "artifact-hashes.json"),
				},
			},
		},
	}); err != nil {
		return Result{}, err
	}

	manifestPath := filepath.Join(opts.OutDir, "baseline-manifest.json")
	manifest := baselineManifest{
		Schema:                 Schema,
		GeneratedAt:            generatedAt,
		GitHead:                strings.TrimSpace(opts.GitHead),
		GitDirty:               strings.TrimSpace(opts.GitStatusShort) != "",
		GitStatusShort:         splitNonEmptyLines(opts.GitStatusShort),
		TargetOS:               runtime.GOOS,
		TargetArch:             runtime.GOARCH,
		AllocatorMode:          allocatorMode,
		WorkloadKind:           "steady_state",
		Iterations:             opts.Iterations,
		WorkBytes:              opts.WorkBytes,
		TelemetrySchema:        rsstelemetry.SchemaV2,
		TelemetryMethod:        rsstelemetry.MethodLinuxProcfsPhaseRSSSamplerV2,
		RSSSidecar:             "rss-telemetry-v2.json",
		HostFingerprint:        "host-fingerprint.json",
		CommandManifest:        "command-manifest.json",
		ValidatorOutput:        "validator-output.txt",
		SmapsRollupArtifacts:   smapsArtifacts,
		ArtifactHashManifest:   "artifact-hashes.json",
		ArtifactHashValidation: "artifact-hashes-validation.txt",
		Notes: []string{
			"P0 baseline only; allocator behavior is unchanged",
			"evidence distinguishes runtime procfs samples from allocation-report estimates",
		},
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return Result{}, err
	}
	return Result{
		OutDir:              opts.OutDir,
		ManifestPath:        manifestPath,
		RSSPath:             rssPath,
		ValidatorOutputPath: validatorOutputPath,
	}, nil
}

func allocateAndTouch(bytes int) [][]byte {
	const chunkSize = 4096
	if bytes < chunkSize {
		bytes = chunkSize
	}
	chunks := (bytes + chunkSize - 1) / chunkSize
	out := make([][]byte, chunks)
	for i := range out {
		out[i] = make([]byte, chunkSize)
		for j := 0; j < len(out[i]); j += 512 {
			out[i][j] = byte(i + j)
		}
	}
	return out
}

type monotonicClock struct {
	now  func() time.Time
	last int64
}

func (c *monotonicClock) next() int64 {
	next := c.now().UnixNano()
	if next <= c.last {
		next = c.last + 1
	}
	c.last = next
	return next
}

func readHostFingerprint() hostFingerprint {
	hostname, _ := os.Hostname()
	return hostFingerprint{
		Schema:        "tetra.ram.p0-baseline.host.v1",
		Hostname:      hostname,
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
		KernelRelease: strings.TrimSpace(readOptionalFile("/proc/sys/kernel/osrelease")),
		OSRelease:     osReleaseName(readOptionalFile("/etc/os-release")),
		CPUModel:      cpuModelName(readOptionalFile("/proc/cpuinfo")),
	}
}

func osReleaseName(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}

func cpuModelName(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func readOptionalFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func splitNonEmptyLines(raw string) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
