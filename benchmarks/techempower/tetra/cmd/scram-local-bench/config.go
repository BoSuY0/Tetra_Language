package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func parseLevels(raw string) ([]benchLevel, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("--levels is required")
	}
	parts := strings.Split(raw, ",")
	levels := make([]benchLevel, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, errors.New("--levels contains an empty entry")
		}
		pair := strings.Split(part, ":")
		if len(pair) != 2 {
			return nil, fmt.Errorf("level %q must be concurrency:connections", part)
		}
		concurrency, err := strconv.Atoi(strings.TrimSpace(pair[0]))
		if err != nil || concurrency <= 0 {
			return nil, fmt.Errorf("level %q has invalid concurrency", part)
		}
		connections, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil || connections <= 0 {
			return nil, fmt.Errorf("level %q has invalid connections", part)
		}
		levels = append(levels, benchLevel{Concurrency: concurrency, Connections: connections})
	}
	return levels, nil
}

func parseEndpointNames(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("--endpoints is required")
	}
	allowed := map[string]bool{
		"db":       true,
		"queries":  true,
		"updates":  true,
		"fortunes": true,
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, errors.New("--endpoints contains an empty entry")
		}
		if !allowed[name] {
			return nil, fmt.Errorf("unsupported endpoint %q", name)
		}
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	return out, nil
}

func parsePositiveIntList(raw string, flagName string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%s is required", flagName)
	}
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("%s contains an empty entry", flagName)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("%s entry %q must be a positive integer", flagName, part)
		}
		values = append(values, value)
	}
	return values, nil
}

func endpointBenchmarkSpecs(names []string) ([]endpointBenchmarkSpec, error) {
	specs := make([]endpointBenchmarkSpec, 0, len(names))
	for _, name := range names {
		switch name {
		case "db":
			specs = append(specs, endpointBenchmarkSpec{Name: "db", Path: "/db", Kind: "single-query", Validate: validateWorldHTTP})
		case "queries":
			specs = append(specs, endpointBenchmarkSpec{Name: "queries", Path: "/queries?queries=2", Kind: "multiple-queries", Validate: validateWorldArrayHTTP})
		case "updates":
			specs = append(specs, endpointBenchmarkSpec{Name: "updates", Path: "/updates?queries=2", Kind: "updates", Validate: validateWorldArrayHTTP})
		case "fortunes":
			specs = append(specs, endpointBenchmarkSpec{Name: "fortunes", Path: "/fortunes", Kind: "fortunes", Validate: validateFortunesHTTP})
		default:
			return nil, fmt.Errorf("unsupported endpoint %q", name)
		}
	}
	return specs, nil
}

func findRepoRoot(start string) (string, error) {
	if start == "" {
		start = "."
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "benchmarks", "techempower", "tetra", "setup-postgres.sql")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find Tetra repo root from %s", start)
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func prepareWorkDir(opt options) (string, func(), error) {
	if opt.WorkDir != "" {
		abs, err := filepath.Abs(opt.WorkDir)
		if err != nil {
			return "", func() {}, err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", func() {}, err
		}
		return abs, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "tetra-techempower-scram-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		if !opt.KeepWorkDir {
			_ = os.RemoveAll(dir)
		}
	}
	return dir, cleanup, nil
}

func defaultEmbeddedPostgresCacheDir() string {
	cacheRoot, err := os.UserCacheDir()
	if err != nil || cacheRoot == "" {
		cacheRoot = os.TempDir()
	}
	return filepath.Join(cacheRoot, "tetra", "embedded-postgres")
}

func buildPlanForMode(profile bool, out string, pkg string) buildPlan {
	args := []string{"build"}
	plan := buildPlan{
		Mode:            "release",
		GoBuildTrimpath: true,
		Stripped:        true,
	}
	if profile {
		plan.Mode = "profile"
		plan.GoBuildTrimpath = false
		plan.Stripped = false
		args = append(args, "-gcflags=all=-N -l")
	} else {
		args = append(args, "-trimpath", "-ldflags=-s -w")
	}
	args = append(args, "-o", out, pkg)
	plan.Args = args
	plan.BuildCommand = "go " + strings.Join(args, " ")
	return plan
}

func buildBinary(ctx context.Context, root string, plan buildPlan) error {
	cmd := exec.CommandContext(ctx, "go", plan.Args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK="+filepath.Join(root, "go.work"))
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, combined.String())
	}
	return nil
}

func freeTCPPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func startSCRAMPostgres(root string, workDir string, cacheDir string, port int) (*embeddedpostgres.EmbeddedPostgres, postgresEvidence, error) {
	runtimePath := filepath.Join(workDir, "pg-runtime")
	dataPath := filepath.Join(workDir, "pg-data")
	cfg := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V16).
		Username("benchmarkdbuser").
		Password("benchmarkdbpass").
		Database("hello_world").
		Port(uint32(port)).
		CachePath(cacheDir).
		RuntimePath(runtimePath).
		DataPath(dataPath).
		StartTimeout(60 * time.Second).
		StartParameters(map[string]string{
			"max_connections":         "256",
			"password_encryption":     "scram-sha-256",
			"log_min_messages":        "warning",
			"log_min_error_statement": "warning",
		}).
		Logger(io.Discard)

	first := embeddedpostgres.NewDatabase(cfg)
	if err := first.Start(); err != nil {
		return nil, postgresEvidence{}, err
	}
	if err := first.Stop(); err != nil {
		return nil, postgresEvidence{}, err
	}
	if err := rewritePGHBAForSCRAM(filepath.Join(dataPath, "pg_hba.conf")); err != nil {
		return nil, postgresEvidence{}, err
	}
	pg := embeddedpostgres.NewDatabase(cfg)
	if err := pg.Start(); err != nil {
		return nil, postgresEvidence{}, err
	}
	info := postgresEvidence{
		Version:        string(embeddedpostgres.V16),
		AuthMethod:     "scram-sha-256",
		Host:           "127.0.0.1",
		Port:           port,
		Database:       "hello_world",
		User:           "benchmarkdbuser",
		MaxConnections: "256",
	}
	_ = root
	return pg, info, nil
}

func rewritePGHBAForSCRAM(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	rewrites := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(line)
		methodIndex := -1
		switch fields[0] {
		case "local":
			if len(fields) >= 4 {
				methodIndex = 3
			}
		case "host", "hostssl", "hostnossl", "hostgssenc", "hostnogssenc":
			if len(fields) >= 5 {
				methodIndex = 4
			}
		}
		if methodIndex < 0 {
			continue
		}
		if fields[methodIndex] != "scram-sha-256" {
			fields[methodIndex] = "scram-sha-256"
			rewrites++
		}
		lines[i] = strings.Join(fields, "\t")
	}
	if rewrites == 0 {
		return errors.New("pg_hba.conf did not contain local/host auth lines to rewrite")
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}

func postgresDSN(port int, user string, password string, database string) string {
	return fmt.Sprintf("host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable", port, user, password, database)
}

func seedPostgres(ctx context.Context, root string, db *sql.DB) error {
	raw, err := os.ReadFile(filepath.Join(root, "benchmarks", "techempower", "tetra", "setup-postgres.sql"))
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, string(raw)); err != nil {
		return err
	}
	return nil
}

func enrichPostgresEvidence(ctx context.Context, db *sql.DB, info *postgresEvidence) error {
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SHOW password_encryption").Scan(&info.PasswordEncryption); err != nil {
		return err
	}
	var verifier string
	if err := db.QueryRowContext(ctx, "SELECT rolpassword FROM pg_authid WHERE rolname=$1", info.User).Scan(&verifier); err != nil {
		return err
	}
	info.VerifierPrefix = verifierPrefix(verifier)
	if info.VerifierPrefix != "SCRAM-SHA-256" {
		return fmt.Errorf("role verifier prefix = %q, want SCRAM-SHA-256", info.VerifierPrefix)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM World").Scan(&info.WorldRows); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM Fortune").Scan(&info.FortuneRows); err != nil {
		return err
	}
	if info.WorldRows != 10000 || info.FortuneRows < 12 {
		return fmt.Errorf("unexpected seed counts: World=%d Fortune=%d", info.WorldRows, info.FortuneRows)
	}
	return nil
}

func verifierPrefix(verifier string) string {
	if idx := strings.Index(verifier, "$"); idx > 0 {
		return verifier[:idx]
	}
	return verifier
}

func startServer(ctx context.Context, root string, appBin string, appPort int, pgPort int, opt options) (*exec.Cmd, *bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, appBin)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), serverEnv(appPort, pgPort, opt)...)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := cmd.Start(); err != nil {
		return nil, &log, err
	}
	return cmd, &log, nil
}

func serverEnv(appPort int, pgPort int, opt options) []string {
	env := []string{
		"TETRA_TE_HOST=127.0.0.1",
		"TETRA_TE_PORT=" + strconv.Itoa(appPort),
		"TETRA_TE_WORKERS=" + strconv.Itoa(opt.Workers),
		"TETRA_TE_PG_HOST=127.0.0.1",
		"TETRA_TE_PG_PORT=" + strconv.Itoa(pgPort),
		"TETRA_TE_PG_USER=benchmarkdbuser",
		"TETRA_TE_PG_DATABASE=hello_world",
		"TETRA_TE_PG_PASSWORD=benchmarkdbpass",
		"TETRA_TE_PG_POOL=" + strconv.Itoa(opt.PoolSize),
	}
	if strings.TrimSpace(opt.PprofAddr) != "" {
		env = append(env, "TETRA_TE_PPROF_ADDR="+opt.PprofAddr)
	}
	return env
}

type shutdownEvidence struct {
	Clean bool
	Error string
}

func stopProcess(cmd *exec.Cmd) shutdownEvidence {
	if cmd == nil || cmd.Process == nil {
		return shutdownEvidence{Clean: true}
	}
	done := make(chan error, 1)
	go func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			return shutdownEvidence{Clean: false, Error: err.Error()}
		}
		return shutdownEvidence{Clean: true}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		err := <-done
		if err != nil {
			return shutdownEvidence{Clean: false, Error: err.Error()}
		}
		return shutdownEvidence{Clean: false, Error: "forced kill after timeout"}
	}
}

func waitForHTTP(ctx context.Context, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return last
}
