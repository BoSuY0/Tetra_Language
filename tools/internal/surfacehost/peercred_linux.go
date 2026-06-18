//go:build linux

package surfacehost

import (
	"net"
	"syscall"
)

func unixPeerPID(conn *net.UnixConn) int {
	if conn == nil {
		return 0
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0
	}
	var pid int
	_ = raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err == nil && cred != nil && cred.Pid > 0 {
			pid = int(cred.Pid)
		}
	})
	return pid
}
