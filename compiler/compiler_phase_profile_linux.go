//go:build linux

package compiler

import (
	"os"
	"strconv"
	"strings"
)

func readCompilerProcessRSSBytes() (uint64, bool) {
	raw, err := os.ReadFile("/proc/self/status")
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
		if err != nil || kb == 0 {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}
