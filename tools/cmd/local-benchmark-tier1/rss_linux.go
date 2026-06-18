//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func readProcessRSSBytes(pid int) (uint64, bool) {
	if pid <= 0 {
		return 0, false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

func processStateMaxRSS(state *os.ProcessState) (uint64, uint64, string, bool) {
	if state == nil {
		return 0, 0, "", false
	}
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil || usage.Maxrss <= 0 {
		return 0, 0, "", false
	}
	raw := uint64(usage.Maxrss)
	return raw, raw * 1024, "kilobytes", true
}
