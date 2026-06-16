package webrt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/internal/toon"
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

	toonResp, ok := router.Route(httprt.Request{
		Method: "GET",
		Path:   "/json",
		Headers: []httprt.Header{
			{Name: "Accept", Value: "text/toon"},
		},
	})
	if !ok {
		t.Fatalf("/json route not mounted for TOON")
	}
	if toonResp.StatusCode != 200 || toonResp.ContentType != "text/toon; charset=utf-8" {
		t.Fatalf("/json TOON response metadata = %#v", toonResp)
	}
	decoded = struct {
		Message string `json:"message"`
	}{}
	decodeTOONPayload(t, toonResp.Body, &decoded)
	if decoded.Message != "Hello, World!" {
		t.Fatalf("/json TOON message = %q", decoded.Message)
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

func TestAcceptExplicitTOON(t *testing.T) {
	cases := []struct {
		header string
		want   bool
	}{
		{header: "", want: false},
		{header: "*/*", want: false},
		{header: "application/json", want: false},
		{header: "text/toon", want: true},
		{header: "application/json, text/toon; charset=utf-8", want: true},
		{header: "text/toon;q=0", want: false},
		{header: "Text/Toon; Q=0.5", want: true},
	}
	for _, tc := range cases {
		if got := acceptExplicitTOON(tc.header); got != tc.want {
			t.Fatalf("acceptExplicitTOON(%q) = %v, want %v", tc.header, got, tc.want)
		}
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

func decodeTOONPayload(t *testing.T, raw []byte, out any) {
	t.Helper()
	jsonRaw, err := toon.ConvertTOONToJSON(raw, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("TOON payload did not decode: %v\n%s", err, raw)
	}
	if err := json.Unmarshal(jsonRaw, out); err != nil {
		t.Fatalf("json.Unmarshal converted TOON: %v\nTOON:\n%s\nJSON:\n%s", err, raw, jsonRaw)
	}
}
