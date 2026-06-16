package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"tetra_language/tools/validators/memoryprod"
)

const commandOutputTailBytes = 64 * 1024

type boundedCommandOutput struct {
	limit   int
	buf     []byte
	written int64
}

func newBoundedCommandOutput(limit int) *boundedCommandOutput {
	return &boundedCommandOutput{limit: limit}
}

func (b *boundedCommandOutput) Write(p []byte) (int, error) {
	n := len(p)
	b.written += int64(n)
	if b.limit <= 0 {
		return n, nil
	}
	if n >= b.limit {
		b.buf = append(b.buf[:0], p[n-b.limit:]...)
		return n, nil
	}
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.limit {
		b.buf = append(b.buf[:0], b.buf[len(b.buf)-b.limit:]...)
	}
	return n, nil
}

func (b *boundedCommandOutput) String() string {
	if b.written > int64(len(b.buf)) {
		return fmt.Sprintf("[output truncated: kept last %d of %d bytes]\n%s", len(b.buf), b.written, string(b.buf))
	}
	return string(b.buf)
}

func (r *smokeRunner) recordProcess(name, kind, path string, res processResult) {
	r.processes = append(r.processes, memoryprod.ProcessReport{
		Name:     name,
		Kind:     kind,
		Path:     path,
		Ran:      true,
		Pass:     res.err == nil && res.exitCode == 0,
		ExitCode: intPtr(res.exitCode),
	})
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) processResult {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	stdout := newBoundedCommandOutput(commandOutputTailBytes)
	stderr := newBoundedCommandOutput(commandOutputTailBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String() + stderr.String())
	if cctx.Err() == context.DeadlineExceeded {
		return processResult{exitCode: -1, output: output, err: cctx.Err()}
	}
	return processResult{exitCode: processExitCode(err), output: output, err: err}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return -int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return -1
}

func intPtr(v int) *int { return &v }
