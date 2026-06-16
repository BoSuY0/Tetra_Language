//go:build !linux

package main

import "os"

func readProcessRSSBytes(pid int) (uint64, bool) {
	return 0, false
}

func processStateMaxRSS(state *os.ProcessState) (uint64, uint64, string, bool) {
	return 0, 0, "", false
}
