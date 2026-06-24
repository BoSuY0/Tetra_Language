package testall

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTestAllRunnerObservationCoversNormalSuccessExpectationFailure(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_SUCCESS_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		reportDir := filepath.Join(root, "report")
		_, _ = runTestAll(
			t,
			root,
			[]string{"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1"},
			"--quick",
			"--json-only",
			"--report-dir",
			reportDir,
		)
		t.Fatalf("synthetic assertion failure")
	}, []string{
		"test_all_invocation_id=",
		"test_all_mode=quick",
		"memory fuzz oracle artifact gate",
		"skip_memory_fuzz_present=1",
		"list_result=skipped_by_explicit_control",
		"report_artifact_manifest",
	})
}

func TestTestAllRunnerObservationCoversExpectedFailureMismatch(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_MISMATCH_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		reportDir := filepath.Join(root, "report")
		out, err := runTestAll(
			t,
			root,
			[]string{"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1"},
			"--quick",
			"--keep-going",
			"--json-only",
			"--report-dir",
			reportDir,
		)
		if err == nil {
			t.Fatalf("expected bounds proof controlled failure\n%s", out)
		}
		assertOnlyTestAllFailedStep(t, decodeTestAllSummary(t, out), testAllUnsafePromotionStep)
	}, []string{
		"test_all_invocation_id=",
		"bounds proof blocker suite",
		"skip_bounds_present=1",
		"list_result=skipped_by_explicit_control",
		"fake_go_trace",
	})
}

func TestTestAllSplitRunnerObservationPreservesStderrEvidence(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_SPLIT_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		reportDir := filepath.Join(root, "report")
		stdout, stderr, err := runTestAllSplit(
			t,
			root,
			[]string{"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1"},
			"--quick",
			"--json-only",
			"--report-dir",
			reportDir,
		)
		if err == nil {
			t.Fatalf("expected unsafe controlled failure\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
		}
		if len(stdout) == 0 {
			t.Fatalf("split stdout should still contain summary")
		}
		t.Fatalf("synthetic split assertion failure stdout=%d stderr=%d", len(stdout), len(stderr))
	}, []string{
		"test_all_invocation_id=",
		"test_all_mode=quick",
		"unsafe promotion blocker suite",
		"split_stderr_preview",
		"skip_unsafe_present=1",
	})
}

func TestTestAllWorkingDirRunnerObservationPreservesEvidence(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_WORKDIR_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		nested := filepath.Join(root, "nested", "cwd")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("create nested cwd: %v", err)
		}
		reportDir := filepath.Join(root, "report")
		_, _ = runTestAllFromWorkingDir(
			t,
			root,
			nested,
			[]string{"TETRA_FAKE_SKIP_HOST_LEAK_LIST=1"},
			"--quick",
			"--json-only",
			"--report-dir",
			reportDir,
		)
		t.Fatalf("synthetic working-dir assertion failure")
	}, []string{
		"test_all_invocation_id=",
		"working_dir_relative=nested/cwd",
		"host leak blocker suite",
		"skip_host_leak_present=1",
	})
}

func TestTestAllRunnerObservationSeparatesSequentialRuns(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_SEQUENTIAL_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		quickReport := filepath.Join(root, "report-quick")
		memoryReport := filepath.Join(root, "report-memory")
		fullReport := filepath.Join(root, "report-full")
		if quickReport == memoryReport || memoryReport == fullReport || quickReport == fullReport {
			t.Fatalf("report dirs must be distinct")
		}
		if out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", quickReport); err != nil {
			t.Fatalf("normal quick should pass: %v\n%s", err, out)
		}
		_, _ = runTestAll(
			t,
			root,
			[]string{"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1"},
			"--quick",
			"--json-only",
			"--report-dir",
			memoryReport,
		)
		if out, err := runTestAll(t, root, nil, "--full", "--json-only", "--report-dir", fullReport); err != nil {
			t.Fatalf("normal full should pass: %v\n%s", err, out)
		}
		t.Fatalf("synthetic sequential assertion failure")
	}, []string{
		"test_all_invocation_id=",
		"test_all_mode=quick",
		"memory fuzz oracle artifact gate",
		"skip_memory_fuzz_present=1",
		"list_result=skipped_by_explicit_control",
	})
}

func TestTestAllRunnerObservationDoesNotRequireAssertionCollector(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_NO_COLLECTOR_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		reportDir := filepath.Join(root, "report")
		_, _ = runTestAll(
			t,
			root,
			[]string{"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1"},
			"--quick",
			"--json-only",
			"--report-dir",
			reportDir,
		)
		t.Fatalf("synthetic assertion failure without manual collector")
	}, []string{
		"test_all_invocation_id=",
		"memory fuzz oracle artifact gate",
		"fake_go_trace",
		"report_artifact_manifest",
	})
}

func TestTestAllRunnerObservationHandlesNonJSONFailure(t *testing.T) {
	runRunnerObservationHelper(t, "TETRA_TEST_ALL_RUNNER_OBS_NONJSON_HELPER", func(t *testing.T) {
		root := testAllFakeRepo(t, false)
		_, _ = runTestAll(t, root, nil, "--report-dir")
		t.Fatalf("synthetic non-json assertion failure")
	}, []string{
		"test_all_invocation_id=",
		"summary_decode_status=failed",
		"run_error=",
		"combined_output_preview",
	})
}

func runRunnerObservationHelper(
	t *testing.T,
	marker string,
	child func(*testing.T),
	want []string,
) {
	t.Helper()
	if os.Getenv(marker) == "1" {
		child(t)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run",
		"^"+t.Name()+"$",
	)
	cmd.Env = append(os.Environ(), marker+"=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("helper subprocess timed out\n%s", out)
	}
	if err == nil {
		t.Fatalf("expected helper subprocess to fail\n%s", out)
	}
	for _, needle := range want {
		if !strings.Contains(string(out), needle) {
			t.Fatalf("helper output missing %q:\n%s", needle, out)
		}
	}
}
