package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestConfigFromEnvDefaults(t *testing.T) {
	cfg, err := configFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("configFromEnv defaults: %v", err)
	}
	if cfg.ListenAddress != [4]byte{0, 0, 0, 0} || cfg.ListenPort != 8080 || cfg.Backlog != 4096 {
		t.Fatalf("listen defaults = %#v", cfg)
	}
	if cfg.Workers != runtime.GOMAXPROCS(0) {
		t.Fatalf("workers default = %d, want GOMAXPROCS", cfg.Workers)
	}
	if cfg.PostgresHost != "tfb-database" || cfg.PostgresPort != 5432 || cfg.PostgresUser != "benchmarkdbuser" || cfg.PostgresDatabase != "hello_world" || cfg.PostgresPassword != "" {
		t.Fatalf("postgres defaults = %#v", cfg)
	}
	if cfg.PostgresPoolSize != 256 || cfg.PostgresDialTimeout != 2*time.Second {
		t.Fatalf("pool/timeout defaults = %#v", cfg)
	}
}

func TestConfigFromEnvOverrides(t *testing.T) {
	env := map[string]string{
		"TETRA_TE_HOST":            "127.0.0.1",
		"TETRA_TE_PORT":            "9090",
		"TETRA_TE_BACKLOG":         "128",
		"TETRA_TE_WORKERS":         "3",
		"TETRA_TE_SERVER_NAME":     "Tetra-Test",
		"TETRA_TE_PG_HOST":         "127.0.0.1",
		"TETRA_TE_PG_PORT":         "15432",
		"TETRA_TE_PG_USER":         "custom",
		"TETRA_TE_PG_DATABASE":     "bench",
		"TETRA_TE_PG_PASSWORD":     "secret",
		"TETRA_TE_PG_POOL":         "4",
		"TETRA_TE_PG_DIAL_TIMEOUT": "1500ms",
	}
	cfg, err := configFromEnv(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("configFromEnv overrides: %v", err)
	}
	if cfg.ListenAddress != [4]byte{127, 0, 0, 1} || cfg.ListenPort != 9090 || cfg.Backlog != 128 || cfg.Workers != 3 || cfg.ServerName != "Tetra-Test" {
		t.Fatalf("listen overrides = %#v", cfg)
	}
	if cfg.PostgresHost != "127.0.0.1" || cfg.PostgresPort != 15432 || cfg.PostgresUser != "custom" || cfg.PostgresDatabase != "bench" || cfg.PostgresPassword != "secret" {
		t.Fatalf("postgres overrides = %#v", cfg)
	}
	if cfg.PostgresPoolSize != 4 || cfg.PostgresDialTimeout != 1500*time.Millisecond {
		t.Fatalf("pool/timeout overrides = %#v", cfg)
	}
}

func TestConfigFromEnvRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "host", env: map[string]string{"TETRA_TE_HOST": "localhost"}},
		{name: "port", env: map[string]string{"TETRA_TE_PORT": "0"}},
		{name: "workers", env: map[string]string{"TETRA_TE_WORKERS": "0"}},
		{name: "pg port", env: map[string]string{"TETRA_TE_PG_PORT": "abc"}},
		{name: "pool", env: map[string]string{"TETRA_TE_PG_POOL": "-1"}},
		{name: "timeout", env: map[string]string{"TETRA_TE_PG_DIAL_TIMEOUT": "soon"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := configFromEnv(func(key string) string { return tc.env[key] })
			if err == nil {
				t.Fatalf("configFromEnv(%s) succeeded, want error", tc.name)
			}
		})
	}
}

func TestRandomWorldIDsStayInTechEmpowerRange(t *testing.T) {
	ids := newRandomWorldIDs(1)
	for i := 0; i < 100; i++ {
		got := ids.Next()
		if got < 1 || got > 10000 {
			t.Fatalf("Next() = %d, want 1..10000", got)
		}
	}
}

func TestBenchmarkConfigDeclaresTechEmpowerEndpoints(t *testing.T) {
	raw, err := os.ReadFile(repoPath("benchmarks", "techempower", "tetra", "benchmark_config.json"))
	if err != nil {
		t.Fatalf("ReadFile benchmark_config.json: %v", err)
	}
	var doc struct {
		Framework string                      `json:"framework"`
		Tests     []map[string]map[string]any `json:"tests"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json.Unmarshal benchmark_config.json: %v", err)
	}
	if doc.Framework != "tetra" || len(doc.Tests) != 1 {
		t.Fatalf("benchmark config header = %#v", doc)
	}
	defaults := doc.Tests[0]["default"]
	expected := map[string]any{
		"json_url":      "/json",
		"db_url":        "/db",
		"query_url":     "/queries?queries=",
		"update_url":    "/updates?queries=",
		"fortune_url":   "/fortunes",
		"plaintext_url": "/plaintext",
		"port":          float64(8080),
		"database":      "Postgres",
		"language":      "Tetra",
	}
	for key, want := range expected {
		if got := defaults[key]; got != want {
			t.Fatalf("benchmark_config %s = %#v, want %#v", key, got, want)
		}
	}
}

func TestLocalBenchmarkArtifactsExist(t *testing.T) {
	for _, rel := range []string{
		filepath.Join("benchmarks", "techempower", "tetra", "Dockerfile"),
		filepath.Join("benchmarks", "techempower", "tetra", "README.md"),
		filepath.Join("benchmarks", "techempower", "tetra", "docker-compose.yml"),
		filepath.Join("benchmarks", "techempower", "tetra", "run-bench.sh"),
		filepath.Join("benchmarks", "techempower", "tetra", "run-full-local.sh"),
		filepath.Join("benchmarks", "techempower", "tetra", "run-local.sh"),
		filepath.Join("benchmarks", "techempower", "tetra", "setup-postgres.sql"),
	} {
		if _, err := os.Stat(repoPath(rel)); err != nil {
			t.Fatalf("expected local benchmark artifact %s: %v", rel, err)
		}
	}
}

func TestPostgresSetupSQLMatchesTechEmpowerSchema(t *testing.T) {
	raw, err := os.ReadFile(repoPath("benchmarks", "techempower", "tetra", "setup-postgres.sql"))
	if err != nil {
		t.Fatalf("ReadFile setup-postgres.sql: %v", err)
	}
	sql := string(raw)
	required := []string{
		`CREATE TABLE World`,
		`id integer PRIMARY KEY`,
		`randomNumber integer NOT NULL`,
		`generate_series(1, 10000)`,
		`CREATE TABLE Fortune`,
		`message varchar(2048) NOT NULL`,
		`fortune: No such file or directory`,
		`<script>alert("This should not be displayed in a browser alert box.");</script>`,
		`フレームワークのベンチマーク`,
	}
	for _, want := range required {
		if !strings.Contains(sql, want) {
			t.Fatalf("setup-postgres.sql missing %q", want)
		}
	}
	if got := strings.Count(sql, "INSERT INTO Fortune"); got != 12 {
		t.Fatalf("Fortune insert count = %d, want 12", got)
	}
}

func TestDockerComposeWiresAppAndPostgres(t *testing.T) {
	raw, err := os.ReadFile(repoPath("benchmarks", "techempower", "tetra", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("ReadFile docker-compose.yml: %v", err)
	}
	compose := string(raw)
	for _, want := range []string{
		"tetra-techempower:",
		"tfb-database:",
		"postgres:16-alpine",
		"./setup-postgres.sql:/docker-entrypoint-initdb.d/001-setup-postgres.sql:ro",
		`POSTGRES_INITDB_ARGS: "--auth-host=scram-sha-256 --auth-local=scram-sha-256"`,
		"password_encryption=scram-sha-256",
		"TETRA_TE_PG_HOST: tfb-database",
		"TETRA_TE_PG_PASSWORD: ${TETRA_TE_PG_PASSWORD:-benchmarkdbpass}",
		"TETRA_TE_WORKERS:",
		"pg_isready",
		"tetra-benchmark:",
		"8080:8080",
	} {
		if !strings.Contains(compose, want) {
			t.Fatalf("docker-compose.yml missing %q", want)
		}
	}
}

func TestLocalRunScriptExportsWorkerCount(t *testing.T) {
	raw, err := os.ReadFile(repoPath("benchmarks", "techempower", "tetra", "run-local.sh"))
	if err != nil {
		t.Fatalf("ReadFile run-local.sh: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		`: "${TETRA_TE_WORKERS:=`,
		`: "${TETRA_TE_PG_PASSWORD:=}"`,
		"export TETRA_TE_WORKERS",
		"export TETRA_TE_PG_PASSWORD",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("run-local.sh missing %q", want)
		}
	}
}

func TestFullLocalScriptHasDockerDaemonPreflight(t *testing.T) {
	raw, err := os.ReadFile(repoPath("benchmarks", "techempower", "tetra", "run-full-local.sh"))
	if err != nil {
		t.Fatalf("ReadFile run-full-local.sh: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"docker info",
		"Docker daemon is not reachable",
		"run-bench.sh",
		"TETRA_TE_BENCH_SKIP_DB=false",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("run-full-local.sh missing %q", want)
		}
	}
}

func repoPath(parts ...string) string {
	all := append([]string{"..", "..", ".."}, parts...)
	return filepath.Join(all...)
}
