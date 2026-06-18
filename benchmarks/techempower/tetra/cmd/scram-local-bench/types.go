package main

import (
	"net/http"
	"time"
)

const matrixSchema = "tetra.techempower.single_query_matrix.v1"

type options struct {
	RootDir             string
	SemanticReportPath  string
	MatrixReportPath    string
	EndpointsRaw        string
	LevelsRaw           string
	WorkerLevelsRaw     string
	Repeats             int
	Duration            time.Duration
	Warmup              time.Duration
	SoakDuration        time.Duration
	SemanticRequests    int
	SemanticConcurrency int
	Workers             int
	PoolSize            int
	KeepWorkDir         bool
	WorkDir             string
	CacheDir            string
	ProfileBuild        bool
	PprofDir            string
	PprofAddr           string
}

type benchLevel struct {
	Concurrency int `json:"concurrency"`
	Connections int `json:"connections"`
}

type matrixReport struct {
	Schema           string               `json:"schema"`
	Status           string               `json:"status"`
	GeneratedAt      string               `json:"generated_at"`
	GeneratedLocalAt string               `json:"generated_local_at"`
	Command          string               `json:"command"`
	Environment      benchmarkEnvironment `json:"environment"`
	Git              gitState             `json:"git"`
	Build            buildEvidence        `json:"build"`
	Postgres         postgresEvidence     `json:"postgres"`
	Server           serverEvidence       `json:"server"`
	Resource         resourceEvidence     `json:"resource"`
	SemanticProbe    []semanticCheck      `json:"semantic_probe"`
	Warmup           *dbRunReport         `json:"warmup,omitempty"`
	Soak             *soakReport          `json:"soak,omitempty"`
	Runs             []dbRunReport        `json:"runs"`
	Summary          matrixSummary        `json:"summary"`
	Artifacts        map[string]string    `json:"artifacts"`
	Limitations      []string             `json:"limitations"`
}

type benchmarkEnvironment struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"go_version"`
	Hostname  string `json:"hostname"`
}

type gitState struct {
	Head           string `json:"head"`
	WorktreeStatus string `json:"worktree_status"`
}

type buildEvidence struct {
	AppBinary       string `json:"app_binary"`
	BenchBinary     string `json:"bench_binary"`
	Mode            string `json:"mode"`
	BuildCommand    string `json:"build_command"`
	GoBuildTrimpath bool   `json:"go_build_trimpath"`
	Stripped        bool   `json:"stripped"`
}

type buildPlan struct {
	Mode            string
	Args            []string
	BuildCommand    string
	GoBuildTrimpath bool
	Stripped        bool
}

type postgresEvidence struct {
	Version            string `json:"version"`
	AuthMethod         string `json:"auth_method"`
	PasswordEncryption string `json:"password_encryption"`
	VerifierPrefix     string `json:"verifier_prefix"`
	WorldRows          int    `json:"world_rows"`
	FortuneRows        int    `json:"fortune_rows"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Database           string `json:"database"`
	User               string `json:"user"`
	MaxConnections     string `json:"max_connections"`
}

type serverEvidence struct {
	BaseURL      string `json:"base_url"`
	Workers      int    `json:"workers"`
	WorkerLevels []int  `json:"worker_levels"`
	PoolSize     int    `json:"pool_size"`
}

type semanticCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	Error    string `json:"error,omitempty"`
}

type resourceEvidence struct {
	Start resourceSnapshot `json:"start"`
	End   resourceSnapshot `json:"end"`
}

type resourceSnapshot struct {
	Timestamp        string  `json:"timestamp"`
	PID              int     `json:"pid"`
	ProcessAlive     bool    `json:"process_alive"`
	RSSKB            int64   `json:"rss_kb"`
	FDCount          int     `json:"fd_count"`
	Threads          int     `json:"threads"`
	TCPConnections   int     `json:"tcp_connections"`
	CPUUserSeconds   float64 `json:"cpu_user_seconds"`
	CPUSystemSeconds float64 `json:"cpu_system_seconds"`
	Goroutines       int     `json:"goroutines,omitempty"`
}

type soakReport struct {
	Endpoint         string           `json:"endpoint"`
	Path             string           `json:"path"`
	Workers          int              `json:"workers"`
	Level            benchLevel       `json:"level"`
	DurationSeconds  float64          `json:"duration_seconds"`
	Requests         int              `json:"requests"`
	Successes        int              `json:"successes"`
	Failures         int              `json:"failures"`
	RPS              float64          `json:"rps"`
	AvgLatencyMS     float64          `json:"avg_latency_ms"`
	P99LatencyMS     float64          `json:"p99_latency_ms"`
	P999LatencyMS    float64          `json:"p999_latency_ms"`
	MaxLatencyMS     float64          `json:"max_latency_ms"`
	FirstHalfAvgMS   float64          `json:"first_half_avg_latency_ms"`
	SecondHalfAvgMS  float64          `json:"second_half_avg_latency_ms"`
	LatencyDriftMS   float64          `json:"latency_drift_ms"`
	ResourceStart    resourceSnapshot `json:"resource_start"`
	ResourceEnd      resourceSnapshot `json:"resource_end"`
	OpenSocketsAfter int              `json:"open_sockets_after_shutdown"`
	ShutdownClean    bool             `json:"shutdown_clean"`
	Validation       string           `json:"validation"`
	Error            string           `json:"error,omitempty"`
}

type dbRunReport struct {
	Endpoint        string           `json:"endpoint"`
	Path            string           `json:"path"`
	Kind            string           `json:"kind"`
	Workers         int              `json:"workers"`
	Level           benchLevel       `json:"level"`
	Repeat          int              `json:"repeat"`
	DurationSeconds float64          `json:"duration_seconds"`
	ElapsedSeconds  float64          `json:"elapsed_seconds"`
	Requests        int              `json:"requests"`
	Successes       int              `json:"successes"`
	Failures        int              `json:"failures"`
	Bytes           int64            `json:"bytes"`
	RPS             float64          `json:"rps"`
	AvgLatencyMS    float64          `json:"avg_latency_ms"`
	P50LatencyMS    float64          `json:"p50_latency_ms"`
	P90LatencyMS    float64          `json:"p90_latency_ms"`
	P95LatencyMS    float64          `json:"p95_latency_ms"`
	P99LatencyMS    float64          `json:"p99_latency_ms"`
	P999LatencyMS   float64          `json:"p999_latency_ms"`
	MaxLatencyMS    float64          `json:"max_latency_ms"`
	Resource        resourceSnapshot `json:"resource"`
	Validation      string           `json:"validation"`
	Error           string           `json:"error,omitempty"`
}

type matrixSummary struct {
	RunCount      int     `json:"run_count"`
	TotalRequests int     `json:"total_requests"`
	TotalFailures int     `json:"total_failures"`
	BestRPS       float64 `json:"best_rps"`
	WorstP99MS    float64 `json:"worst_p99_ms"`
	WorstP999MS   float64 `json:"worst_p999_ms"`
	Decision      string  `json:"decision"`
}

type loadResult struct {
	status  int
	bytes   int
	latency time.Duration
	err     error
}

type pprofArtifacts struct {
	CPUProfile  string
	HeapProfile string
}

type endpointBenchmarkSpec struct {
	Name     string
	Path     string
	Kind     string
	Validate func(*http.Response, []byte) error
}

type world struct {
	ID           int `json:"id"`
	RandomNumber int `json:"randomNumber"`
}
