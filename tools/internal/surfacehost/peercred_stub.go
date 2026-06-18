//go:build !linux

package surfacehost

import "net"

func unixPeerPID(conn *net.UnixConn) int {
	return 0
}
