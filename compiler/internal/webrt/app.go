package webrt

import (
	"errors"

	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/pgrt"
)

var (
	ErrMissingRouter       = errors.New("TechEmpower router is nil")
	ErrMissingPostgresPool = errors.New("TechEmpower app requires PostgreSQL pool")
)

type TechEmpowerRoutes struct {
	Pool       *pgrt.Pool
	NextID     func() int
	NextRandom func() int
}

type TechEmpowerServerConfig struct {
	Address    [4]byte
	Port       int
	Backlog    int
	ServerName string
	DateFunc   func() string
	Pool       *pgrt.Pool
	NextID     func() int
	NextRandom func() int
}

func RegisterTechEmpowerRoutes(router *httprt.Router, cfg TechEmpowerRoutes) error {
	if router == nil {
		return ErrMissingRouter
	}
	if cfg.Pool == nil {
		return ErrMissingPostgresPool
	}
	router.Handle("GET", "/plaintext", PlaintextHandler())
	router.Handle("GET", "/json", JSONMessageHandler("Hello, World!"))
	router.Handle("GET", "/db", DBHandler(cfg.Pool, cfg.NextID))
	router.Handle("GET", "/queries", QueriesHandler(cfg.Pool, cfg.NextID))
	router.Handle("GET", "/updates", UpdatesHandler(cfg.Pool, cfg.NextID, cfg.NextRandom))
	router.Handle("GET", "/fortunes", FortunesHandler(cfg.Pool))
	return nil
}

func NewTechEmpowerServer(cfg TechEmpowerServerConfig) (*Server, error) {
	serverName := cfg.ServerName
	if serverName == "" {
		serverName = "Tetra-TechEmpower"
	}
	srv := NewServer(Config{
		Address:    cfg.Address,
		Port:       cfg.Port,
		Backlog:    cfg.Backlog,
		ServerName: serverName,
		DateFunc:   cfg.DateFunc,
	})
	if err := RegisterTechEmpowerRoutes(&srv.Router, TechEmpowerRoutes{
		Pool:       cfg.Pool,
		NextID:     cfg.NextID,
		NextRandom: cfg.NextRandom,
	}); err != nil {
		return nil, err
	}
	return srv, nil
}

func PlaintextHandler() httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		return httprt.Response{
			StatusCode:  200,
			ContentType: "text/plain",
			Body:        []byte("Hello, World!"),
		}
	}
}
