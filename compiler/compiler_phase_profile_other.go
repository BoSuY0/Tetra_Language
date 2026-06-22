//go:build !linux

package compiler

func readCompilerProcessRSSBytes() (uint64, bool) {
	return 0, false
}
