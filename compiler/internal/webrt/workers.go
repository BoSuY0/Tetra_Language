package webrt

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type WorkerFactory func(workerID int, port int) (*Server, error)

type WorkerGroup struct {
	servers []*Server
}

func ListenWorkers(count int, initialPort int, factory WorkerFactory) (*WorkerGroup, error) {
	if count <= 0 {
		count = 1
	}
	if factory == nil {
		return nil, errors.New("worker factory is nil")
	}
	group := &WorkerGroup{servers: make([]*Server, 0, count)}
	port := initialPort
	for workerID := 0; workerID < count; workerID++ {
		server, err := factory(workerID, port)
		if err != nil {
			_ = group.Close()
			return nil, err
		}
		if server == nil {
			_ = group.Close()
			return nil, fmt.Errorf("worker %d server is nil", workerID)
		}
		if err := server.Listen(); err != nil {
			_ = group.Close()
			return nil, fmt.Errorf("worker %d listen: %w", workerID, err)
		}
		if workerID == 0 {
			port = server.Port()
		}
		group.servers = append(group.servers, server)
	}
	return group, nil
}

func (g *WorkerGroup) Count() int {
	if g == nil {
		return 0
	}
	return len(g.servers)
}

func (g *WorkerGroup) Port() int {
	if g == nil || len(g.servers) == 0 {
		return 0
	}
	return g.servers[0].Port()
}

func (g *WorkerGroup) Ports() []int {
	if g == nil {
		return nil
	}
	ports := make([]int, 0, len(g.servers))
	for _, server := range g.servers {
		ports = append(ports, server.Port())
	}
	return ports
}

func (g *WorkerGroup) Serve(parent context.Context) error {
	if g == nil || len(g.servers) == 0 {
		return errors.New("worker group has no servers")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	var wg sync.WaitGroup
	errc := make(chan error, len(g.servers))
	for _, server := range g.servers {
		server := server
		wg.Add(1)
		go func() {
			defer wg.Done()
			errc <- server.Serve(ctx)
		}()
	}
	go func() {
		wg.Wait()
		close(errc)
	}()

	var firstErr error
	for err := range errc {
		if err == nil || errors.Is(err, context.Canceled) {
			continue
		}
		if firstErr == nil {
			firstErr = err
			cancel()
			_ = g.Close()
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return parent.Err()
}

func (g *WorkerGroup) Close() error {
	if g == nil {
		return nil
	}
	var firstErr error
	for _, server := range g.servers {
		if err := server.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
