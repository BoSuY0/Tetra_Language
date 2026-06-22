//go:build linux

package rsstelemetry

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func ReadProcessRSSBytes(pid int) (uint64, bool) {
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

func ProcessStateMaxRSS(state *os.ProcessState) (uint64, uint64, string, bool) {
	if state == nil {
		return 0, 0, "", false
	}
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil || usage.Maxrss <= 0 {
		return 0, 0, "", false
	}
	raw := uint64(usage.Maxrss)
	return raw, raw * 1024, UnitKilobytes, true
}

func ReadProcessMappingCount(pid int) (uint64, bool) {
	if pid <= 0 {
		return 0, false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/maps")
	if err != nil || len(raw) == 0 {
		return 0, false
	}
	count := uint64(bytes.Count(raw, []byte{'\n'}))
	if raw[len(raw)-1] != '\n' {
		count++
	}
	if count == 0 {
		return 0, false
	}
	return count, true
}

func ReadProcessSmapsRollup(pid int) ([]byte, bool) {
	if pid <= 0 {
		return nil, false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/smaps_rollup")
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	return raw, true
}
