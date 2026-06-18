//go:build !linux

package surfacehost

import "fmt"

func NewWaylandBackend() (Backend, error) {
	return nil, fmt.Errorf("Wayland Surface host is only available on linux")
}
