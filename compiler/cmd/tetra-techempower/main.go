package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"tetra_language/compiler/internal/pgrt"
	"tetra_language/compiler/internal/webrt"
)

type appConfig struct {
	ListenAddress       [4]byte
	ListenPort          int
	Backlog             int
	Workers             int
	ServerName          string
	PostgresHost        string
	PostgresPort        int
	PostgresUser        string
	PostgresDatabase    string
	PostgresPassword    string
	PostgresPoolSize    int
	PostgresDialTimeout time.Duration
	PprofAddr           string
}

type randomWorldIDs struct {
	mu sync.Mutex
	r  *rand.Rand
}

func main() {
	cfg, err := configFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := serve(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(parent context.Context, cfg appConfig) error {
	ids := newRandomWorldIDs(time.Now().UnixNano())
	address := net.JoinHostPort(cfg.PostgresHost, strconv.Itoa(cfg.PostgresPort))
	pool, err := pgrt.NewDialPool(cfg.PostgresPoolSize, pgrt.DialConfig{
		Network: "tcp",
		Address: address,
		Timeout: cfg.PostgresDialTimeout,
		Startup: pgrt.StartupConfig{
			User:     cfg.PostgresUser,
			Database: cfg.PostgresDatabase,
			Password: cfg.PostgresPassword,
			Parameters: map[string]string{
				"application_name": "tetra-techempower",
				"client_encoding":  "UTF8",
			},
		},
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	workers, err := webrt.ListenWorkers(
		cfg.Workers,
		cfg.ListenPort,
		func(_ int, port int) (*webrt.Server, error) {
			return webrt.NewTechEmpowerServer(webrt.TechEmpowerServerConfig{
				Address:    cfg.ListenAddress,
				Port:       port,
				Backlog:    cfg.Backlog,
				ServerName: cfg.ServerName,
				Pool:       pool,
				NextID:     ids.Next,
				NextRandom: ids.Next,
			})
		},
	)
	if err != nil {
		return err
	}
	defer workers.Close()
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	pprofServer, err := startPprofServer(ctx, cfg.PprofAddr)
	if err != nil {
		return err
	}
	defer shutdownPprofServer(pprofServer)

	fmt.Fprintf(
		os.Stderr,
		"tetra-techempower listening on %d with %d workers\n",
		workers.Port(),
		workers.Count(),
	)
	err = workers.Serve(ctx)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func configFromEnv(getenv func(string) string) (appConfig, error) {
	host, err := parseIPv4(envOr(getenv, "TETRA_TE_HOST", "0.0.0.0"))
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_HOST: %w", err)
	}
	port, err := parsePositiveInt(envOr(getenv, "TETRA_TE_PORT", "8080"), 65535)
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_PORT: %w", err)
	}
	backlog, err := parsePositiveInt(envOr(getenv, "TETRA_TE_BACKLOG", "4096"), 1<<20)
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_BACKLOG: %w", err)
	}
	workerDefault := runtime.GOMAXPROCS(0)
	workers, err := parsePositiveInt(
		envOr(getenv, "TETRA_TE_WORKERS", strconv.Itoa(workerDefault)),
		1<<20,
	)
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_WORKERS: %w", err)
	}
	pgPort, err := parsePositiveInt(envOr(getenv, "TETRA_TE_PG_PORT", "5432"), 65535)
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_PG_PORT: %w", err)
	}
	poolSizeDefault := 256
	poolSize, err := parsePositiveInt(
		envOr(getenv, "TETRA_TE_PG_POOL", strconv.Itoa(poolSizeDefault)),
		1<<20,
	)
	if err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_PG_POOL: %w", err)
	}
	timeout, err := time.ParseDuration(envOr(getenv, "TETRA_TE_PG_DIAL_TIMEOUT", "2s"))
	if err != nil || timeout <= 0 {
		return appConfig{}, fmt.Errorf("TETRA_TE_PG_DIAL_TIMEOUT: must be a positive duration")
	}
	pprofAddr := getenv("TETRA_TE_PPROF_ADDR")
	if err := validatePprofAddr(pprofAddr); err != nil {
		return appConfig{}, fmt.Errorf("TETRA_TE_PPROF_ADDR: %w", err)
	}
	return appConfig{
		ListenAddress:       host,
		ListenPort:          port,
		Backlog:             backlog,
		Workers:             workers,
		ServerName:          envOr(getenv, "TETRA_TE_SERVER_NAME", "Tetra-TechEmpower"),
		PostgresHost:        envOr(getenv, "TETRA_TE_PG_HOST", "tfb-database"),
		PostgresPort:        pgPort,
		PostgresUser:        envOr(getenv, "TETRA_TE_PG_USER", "benchmarkdbuser"),
		PostgresDatabase:    envOr(getenv, "TETRA_TE_PG_DATABASE", "hello_world"),
		PostgresPassword:    getenv("TETRA_TE_PG_PASSWORD"),
		PostgresPoolSize:    poolSize,
		PostgresDialTimeout: timeout,
		PprofAddr:           pprofAddr,
	}, nil
}

func validatePprofAddr(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return nil
	}
	host, portRaw, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if _, err := parsePositiveInt(portRaw, 65535); err != nil {
		return err
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("%q must bind to localhost or a loopback IP", addr)
	}
	return nil
}

func newPprofHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func startPprofServer(ctx context.Context, addr string) (*http.Server, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, nil
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: newPprofHandler()}
	go func() {
		<-ctx.Done()
		shutdownPprofServer(server)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "tetra-techempower pprof server error: %v\n", err)
		}
	}()
	fmt.Fprintf(os.Stderr, "tetra-techempower pprof listening on %s\n", listener.Addr().String())
	return server, nil
}

func shutdownPprofServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func envOr(getenv func(string) string, key string, fallback string) string {
	value := getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseIPv4(value string) ([4]byte, error) {
	parsed := net.ParseIP(value).To4()
	if parsed == nil {
		return [4]byte{}, fmt.Errorf("%q is not an IPv4 address", value)
	}
	return [4]byte{parsed[0], parsed[1], parsed[2], parsed[3]}, nil
}

func parsePositiveInt(value string, max int) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("%q must be in range 1..%d", value, max)
	}
	return n, nil
}

func newRandomWorldIDs(seed int64) *randomWorldIDs {
	return &randomWorldIDs{r: rand.New(rand.NewSource(seed))}
}

func (ids *randomWorldIDs) Next() int {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	return ids.r.Intn(10000) + 1
}
