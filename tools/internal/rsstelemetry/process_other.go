//go:build !linux

package rsstelemetry

import "os"

func ReadProcessRSSBytes(pid int) (uint64, bool) {
	return 0, false
}

func ProcessStateMaxRSS(state *os.ProcessState) (uint64, uint64, string, bool) {
	return 0, 0, "", false
}
