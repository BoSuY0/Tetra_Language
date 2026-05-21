package pgrt

import (
	"context"
	"errors"
	"net"
	"time"
)

var ErrMissingAddress = errors.New("PostgreSQL dial address is empty")

type DialConfig struct {
	Network string
	Address string
	Timeout time.Duration
	Startup StartupConfig
}

func Dial(ctx context.Context, cfg DialConfig) (*Conn, error) {
	if cfg.Address == "" {
		return nil, ErrMissingAddress
	}
	network := cfg.Network
	if network == "" {
		network = "tcp"
	}
	dialer := net.Dialer{}
	if cfg.Timeout > 0 {
		dialer.Timeout = cfg.Timeout
	}
	rwc, err := dialer.DialContext(ctx, network, cfg.Address)
	if err != nil {
		return nil, err
	}
	conn, err := Connect(ctx, rwc, cfg.Startup)
	if err != nil {
		_ = rwc.Close()
		return nil, err
	}
	return conn, nil
}

func DialConnector(cfg DialConfig) Connector {
	return func(ctx context.Context) (*Conn, error) {
		return Dial(ctx, cfg)
	}
}

func NewDialPool(maxOpen int, cfg DialConfig) (*Pool, error) {
	return NewPool(maxOpen, DialConnector(cfg))
}
