package webrt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/pgrt"
)

func TestRegisterTechEmpowerRoutesMountsAllEndpoints(t *testing.T) {
	pool, err := pgrt.NewPool(1, func(ctx context.Context) (*pgrt.Conn, error) {
		return nil, errors.New("db intentionally unavailable")
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	var router httprt.Router
	if err := RegisterTechEmpowerRoutes(&router, TechEmpowerRoutes{Pool: pool, NextID: sequenceIDs(1), NextRandom: sequenceIDs(2)}); err != nil {
		t.Fatalf("RegisterTechEmpowerRoutes: %v", err)
	}

	plaintext, ok := router.Route(httprt.Request{Method: "GET", Path: "/plaintext"})
	if !ok {
		t.Fatalf("/plaintext route not mounted")
	}
	if plaintext.StatusCode != 200 || plaintext.ContentType != "text/plain" || string(plaintext.Body) != "Hello, World!" {
		t.Fatalf("/plaintext response = %#v", plaintext)
	}

	jsonResp, ok := router.Route(httprt.Request{Method: "GET", Path: "/json"})
	if !ok {
		t.Fatalf("/json route not mounted")
	}
	if jsonResp.StatusCode != 200 || jsonResp.ContentType != "application/json" {
		t.Fatalf("/json response metadata = %#v", jsonResp)
	}
	var decoded struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(jsonResp.Body, &decoded); err != nil {
		t.Fatalf("json.Unmarshal /json: %v", err)
	}
	if decoded.Message != "Hello, World!" {
		t.Fatalf("/json message = %q", decoded.Message)
	}

	for _, path := range []string{"/db", "/queries", "/updates", "/fortunes"} {
		resp, ok := router.Route(httprt.Request{Method: "GET", Path: path})
		if !ok {
			t.Fatalf("%s route not mounted", path)
		}
		if resp.StatusCode != 500 {
			t.Fatalf("%s synthetic DB failure status = %d, want 500", path, resp.StatusCode)
		}
	}
}

func TestRegisterTechEmpowerRoutesRequiresPool(t *testing.T) {
	var router httprt.Router
	err := RegisterTechEmpowerRoutes(&router, TechEmpowerRoutes{})
	if !errors.Is(err, ErrMissingPostgresPool) {
		t.Fatalf("RegisterTechEmpowerRoutes without pool = %v, want ErrMissingPostgresPool", err)
	}
}

func TestNewTechEmpowerServerAppliesConfigAndRoutes(t *testing.T) {
	pool, err := pgrt.NewPool(1, func(ctx context.Context) (*pgrt.Conn, error) {
		return nil, errors.New("db intentionally unavailable")
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	srv, err := NewTechEmpowerServer(TechEmpowerServerConfig{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       8080,
		Backlog:    128,
		ServerName: "Tetra-TechEmpower-Test",
		Pool:       pool,
		NextID:     sequenceIDs(1),
		NextRandom: sequenceIDs(2),
	})
	if err != nil {
		t.Fatalf("NewTechEmpowerServer: %v", err)
	}
	if srv.Address != [4]byte{127, 0, 0, 1} || srv.Port() != 0 || srv.Config.Port != 8080 || srv.Backlog != 128 || srv.ServerName != "Tetra-TechEmpower-Test" {
		t.Fatalf("server config = %#v", srv.Config)
	}
	if _, ok := srv.Router.Route(httprt.Request{Method: "GET", Path: "/plaintext"}); !ok {
		t.Fatalf("NewTechEmpowerServer did not mount /plaintext")
	}
}
