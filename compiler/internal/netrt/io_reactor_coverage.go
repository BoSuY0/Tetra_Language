package netrt

import (
	"fmt"
	"strings"
)

type IOReactorEvidenceID string

const (
	IOReactorLinuxEpollV1               IOReactorEvidenceID = "linux_epoll_v1"
	IOReactorIOUringFuture              IOReactorEvidenceID = "io_uring_future"
	IOReactorKqueueBoundary             IOReactorEvidenceID = "kqueue_macos_boundary"
	IOReactorIOCPBoundary               IOReactorEvidenceID = "iocp_windows_boundary"
	IOReactorWASIWebBoundary            IOReactorEvidenceID = "wasi_web_adapter_boundary"
	IOReactorNonblockingAcceptReadWrite IOReactorEvidenceID = "nonblocking_accept_read_write"
	IOReactorReadinessPolling           IOReactorEvidenceID = "readiness_polling"
	IOReactorTaskWakeupsFromIO          IOReactorEvidenceID = "task_wakeups_from_io"
	IOReactorTimerIntegration           IOReactorEvidenceID = "timer_integration"
	IOReactorCancellation               IOReactorEvidenceID = "cancellation"
	IOReactorBackpressure               IOReactorEvidenceID = "backpressure"
	IOReactorReportRows                 IOReactorEvidenceID = "reactor_report_rows"
	IOReactorHTTPSmoke                  IOReactorEvidenceID = "http_smoke"
	IOReactorDBSmoke                    IOReactorEvidenceID = "db_smoke"
	IOReactorStressEvidence             IOReactorEvidenceID = "stress_evidence"
)

type IOReactorEvidenceStatus string

const (
	IOReactorImplementedNarrow  IOReactorEvidenceStatus = "implemented_narrow"
	IOReactorBoundaryDocumented IOReactorEvidenceStatus = "boundary_documented"
	IOReactorEvidenceOnly       IOReactorEvidenceStatus = "evidence_only"
)

type IOReactorCoverageReport struct {
	SchemaVersion                 string                 `json:"schema_version"`
	Rows                          []IOReactorEvidenceRow `json:"rows"`
	NonClaims                     []string               `json:"non_claims"`
	FullProductionWebStackClaimed bool                   `json:"full_production_web_stack_claimed"`
	CrossPlatformParityClaimed    bool                   `json:"cross_platform_parity_claimed"`
	IOUringClaimed                bool                   `json:"io_uring_claimed"`
	RuntimeBehaviorChanged        bool                   `json:"runtime_behavior_changed"`
}

type IOReactorEvidenceRow struct {
	ID                           IOReactorEvidenceID     `json:"id"`
	Name                         string                  `json:"name"`
	Status                       IOReactorEvidenceStatus `json:"status"`
	Platform                     string                  `json:"platform,omitempty"`
	RequiredFacts                []string                `json:"required_facts,omitempty"`
	MissingFacts                 []string                `json:"missing_facts,omitempty"`
	Evidence                     string                  `json:"evidence"`
	Boundary                     string                  `json:"boundary"`
	ClaimsFullProductionWebStack bool                    `json:"claims_full_production_web_stack,omitempty"`
	ClaimsCrossPlatformParity    bool                    `json:"claims_cross_platform_parity,omitempty"`
	ClaimsIOUring                bool                    `json:"claims_io_uring,omitempty"`
	ClaimsRuntimeBehaviorChange  bool                    `json:"claims_runtime_behavior_change,omitempty"`
}

func IOReactorCoverage() (IOReactorCoverageReport, error) {
	return IOReactorCoverageReport{
		SchemaVersion: "tetra.runtime.io_reactor.v1",
		Rows: []IOReactorEvidenceRow{
			linuxEpollV1Row(),
			ioUringFutureRow(),
			kqueueBoundaryRow(),
			iocpBoundaryRow(),
			wasiWebBoundaryRow(),
			nonblockingAcceptReadWriteRow(),
			readinessPollingRow(),
			taskWakeupsFromIORow(),
			timerIntegrationRow(),
			cancellationRow(),
			backpressureRow(),
			reactorReportRowsRow(),
			httpSmokeRow(),
			dbSmokeRow(),
			stressEvidenceRow(),
		},
		NonClaims: []string{
			"full production web stack is not claimed",
			"cross-platform reactor parity is not claimed",
			"io_uring is not implemented",
			"runtime behavior is unchanged by this report-only P18.3 coverage",
			"macOS kqueue, Windows IOCP, and WASI/web event adapters remain documented platform boundaries",
		},
		FullProductionWebStackClaimed: false,
		CrossPlatformParityClaimed:    false,
		IOUringClaimed:                false,
		RuntimeBehaviorChanged:        false,
	}, nil
}

func ValidateIOReactorCoverage(report IOReactorCoverageReport) error {
	if report.SchemaVersion != "tetra.runtime.io_reactor.v1" {
		return fmt.Errorf("io reactor coverage: schema = %q", report.SchemaVersion)
	}
	if report.FullProductionWebStackClaimed {
		return fmt.Errorf(
			"io reactor coverage: full production web stack claim is forbidden for P18.3",
		)
	}
	if report.CrossPlatformParityClaimed {
		return fmt.Errorf(
			"io reactor coverage: cross-platform reactor parity claim is forbidden for P18.3",
		)
	}
	if report.IOUringClaimed {
		return fmt.Errorf("io reactor coverage: io_uring claim is forbidden until a later slice")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf(
			"io reactor coverage: runtime behavior change claim is forbidden for report-only P18.3 evidence",
		)
	}
	for _, want := range []string{
		"full production web stack is not claimed",
		"cross-platform reactor parity is not claimed",
		"io_uring is not implemented",
		"runtime behavior is unchanged",
	} {
		if !containsIOReactorText(report.NonClaims, want) {
			return fmt.Errorf("io reactor coverage: missing non-claim %q", want)
		}
	}

	expectedStatus := map[IOReactorEvidenceID]IOReactorEvidenceStatus{
		IOReactorLinuxEpollV1:               IOReactorImplementedNarrow,
		IOReactorIOUringFuture:              IOReactorBoundaryDocumented,
		IOReactorKqueueBoundary:             IOReactorBoundaryDocumented,
		IOReactorIOCPBoundary:               IOReactorBoundaryDocumented,
		IOReactorWASIWebBoundary:            IOReactorBoundaryDocumented,
		IOReactorNonblockingAcceptReadWrite: IOReactorImplementedNarrow,
		IOReactorReadinessPolling:           IOReactorImplementedNarrow,
		IOReactorTaskWakeupsFromIO:          IOReactorImplementedNarrow,
		IOReactorTimerIntegration:           IOReactorEvidenceOnly,
		IOReactorCancellation:               IOReactorEvidenceOnly,
		IOReactorBackpressure:               IOReactorEvidenceOnly,
		IOReactorReportRows:                 IOReactorEvidenceOnly,
		IOReactorHTTPSmoke:                  IOReactorEvidenceOnly,
		IOReactorDBSmoke:                    IOReactorEvidenceOnly,
		IOReactorStressEvidence:             IOReactorEvidenceOnly,
	}
	if len(report.Rows) != len(expectedStatus) {
		return fmt.Errorf(
			"io reactor coverage: row count = %d, want %d",
			len(report.Rows),
			len(expectedStatus),
		)
	}
	rows := map[IOReactorEvidenceID]IOReactorEvidenceRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("io reactor coverage: row missing id")
		}
		wantStatus, ok := expectedStatus[row.ID]
		if !ok {
			return fmt.Errorf("io reactor coverage: unexpected row %q", row.ID)
		}
		if _, exists := rows[row.ID]; exists {
			return fmt.Errorf("io reactor coverage: duplicate row %q", row.ID)
		}
		rows[row.ID] = row
		if row.Status != wantStatus {
			return fmt.Errorf(
				"io reactor coverage: row %q status = %q, want %q",
				row.ID,
				row.Status,
				wantStatus,
			)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("io reactor coverage: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("io reactor coverage: row %q missing required facts", row.ID)
		}
		if row.ClaimsFullProductionWebStack {
			return fmt.Errorf(
				"io reactor coverage: row %q claims full production web stack",
				row.ID,
			)
		}
		if row.ClaimsCrossPlatformParity {
			return fmt.Errorf(
				"io reactor coverage: row %q claims cross-platform reactor parity",
				row.ID,
			)
		}
		if row.ClaimsIOUring {
			return fmt.Errorf("io reactor coverage: row %q claims io_uring support", row.ID)
		}
		if row.ClaimsRuntimeBehaviorChange {
			return fmt.Errorf("io reactor coverage: row %q claims runtime behavior change", row.ID)
		}
	}
	for id := range expectedStatus {
		if _, ok := rows[id]; !ok {
			return fmt.Errorf("io reactor coverage: missing row %q", id)
		}
	}

	if err := requireIOReactorCoverageFacts(
		rows[IOReactorLinuxEpollV1],
		"linux epoll v1",
		"epoll v1",
		"NewPoller",
		"EPOLLIN",
		"EPOLLOUT",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorIOUringFuture],
		"io_uring",
		"io_uring later after epoll stable",
		"not implemented",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorKqueueBoundary],
		"kqueue",
		"kqueue macOS",
		"not implemented",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorIOCPBoundary],
		"IOCP",
		"IOCP Windows",
		"not implemented",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorWASIWebBoundary],
		"WASI/web",
		"WASI/web event adapters",
		"not implemented",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorNonblockingAcceptReadWrite],
		"nonblocking accept/read/write",
		"Accept4",
		"SOCK_NONBLOCK",
		"Read",
		"Write",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorReadinessPolling],
		"readiness polling",
		"Poller.Wait",
		"wait-one readiness",
		"TestNetRuntimeEpollReadiness",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorTaskWakeupsFromIO],
		"I/O task wakeups",
		"I/O readiness wakes",
		"poller.Wait",
		"core.task_spawn_i32",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorTimerIntegration],
		"timer integration",
		"50ms poll timeout",
		"sleep_ms",
		"wake in deadline order",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorCancellation],
		"cancellation",
		"context cancellation",
		"Close",
		"task_group_cancel",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorBackpressure],
		"backpressure",
		"EPOLLOUT",
		"ErrPoolExhausted",
		"output buffer",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorReportRows],
		"reactor report rows",
		"tetra.techempower.benchmark.v1",
		"ValidateReport",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorHTTPSmoke],
		"HTTP smoke",
		"plaintext",
		"json",
		"pipelining",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorDBSmoke],
		"DB smoke",
		"PostgreSQL",
		"prepared statement",
		"pool",
	); err != nil {
		return err
	}
	if err := requireIOReactorCoverageFacts(
		rows[IOReactorStressEvidence],
		"stress evidence",
		"go test -race ./compiler/internal/netrt",
		"many readiness waits",
	); err != nil {
		return err
	}
	return nil
}

func linuxEpollV1Row() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorLinuxEpollV1,
		Name:     "Linux epoll v1",
		Status:   IOReactorImplementedNarrow,
		Platform: "linux",
		RequiredFacts: []string{
			("epoll v1 is implemented by NewPoller, Poller.AddRead, " +
				"Poller.AddReadWrite, Poller.Mod, Poller.Remove, and " +
				"Poller.Wait"),
			"events capture EPOLLIN, EPOLLOUT, EPOLLERR, and EPOLLHUP readiness flags",
			("compiler/internal/netrt/netrt_linux_test.go::" +
				"TestPollerSignalsReadableDataAndSyscallReadWriteRoundTrip " +
				"covers Linux epoll readiness"),
		},
		Evidence: ("compiler/internal/netrt/netrt_linux.go::NewPoller; " +
			"compiler/internal/netrt/netrt_linux.go::Poller.Wait; " +
			"compiler/internal/netrt/netrt_linux_test.go::" +
			"TestPollerSignalsReadableDataAndSyscallReadWriteRoundTrip"),
		Boundary: ("Linux epoll v1 is implemented narrowly for current netrt " +
			"sockets; this does not implement io_uring or a portable " +
			"event-loop abstraction"),
	}
}

func ioUringFutureRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorIOUringFuture,
		Name:     "io_uring future boundary",
		Status:   IOReactorBoundaryDocumented,
		Platform: "linux",
		RequiredFacts: []string{
			"io_uring later after epoll stable is the P18.3 master-plan boundary",
			"io_uring is not implemented in netrt",
			"no row may claim io_uring support in P18.3",
		},
		MissingFacts: []string{
			"io_uring submit/completion queue implementation",
			"io_uring cancellation and backpressure tests",
			"io_uring parity with epoll evidence",
		},
		Evidence: ("/home/tetra/Downloads/tetra_ideal_master_plan_20260602.md::" +
			"P18.3; compiler/internal/netrt/netrt_linux.go::NewPoller"),
		Boundary: ("io_uring remains future work until a separate evidence " +
			"slice implements and validates it"),
	}
}

func kqueueBoundaryRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorKqueueBoundary,
		Name:     "macOS kqueue boundary",
		Status:   IOReactorBoundaryDocumented,
		Platform: "macos",
		RequiredFacts: []string{
			"kqueue macOS is documented in the P18.3 cross-platform path",
			"kqueue is not implemented in netrt",
			"compiler/internal/netrt/netrt_unsupported.go returns ErrUnsupported outside Linux",
		},
		MissingFacts: []string{
			"kqueue poller implementation",
			"macOS nonblocking socket runtime smoke",
			"macOS cancellation/backpressure validation",
		},
		Evidence: ("/home/tetra/Downloads/tetra_ideal_master_plan_20260602.md::" +
			"P18.3; compiler/internal/netrt/netrt_unsupported.go"),
		Boundary: ("macOS kqueue support is a documented platform boundary, not " +
			"implemented reactor parity"),
	}
}

func iocpBoundaryRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorIOCPBoundary,
		Name:     "Windows IOCP boundary",
		Status:   IOReactorBoundaryDocumented,
		Platform: "windows",
		RequiredFacts: []string{
			"IOCP Windows is documented in the P18.3 cross-platform path",
			"IOCP is not implemented in netrt",
			"compiler/internal/netrt/netrt_unsupported.go returns ErrUnsupported outside Linux",
		},
		MissingFacts: []string{
			"IOCP completion port implementation",
			"Windows socket runtime smoke",
			"Windows cancellation/backpressure validation",
		},
		Evidence: ("/home/tetra/Downloads/tetra_ideal_master_plan_20260602.md::" +
			"P18.3; compiler/internal/netrt/netrt_unsupported.go"),
		Boundary: ("Windows IOCP support is a documented platform boundary, not " +
			"implemented reactor parity"),
	}
}

func wasiWebBoundaryRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorWASIWebBoundary,
		Name:     "WASI/web event adapter boundary",
		Status:   IOReactorBoundaryDocumented,
		Platform: "wasi-web",
		RequiredFacts: []string{
			"WASI/web event adapters are documented in the P18.3 cross-platform path",
			"WASI/web event adapters are not implemented in netrt",
			("non-Linux netrt calls return ErrUnsupported through " +
				"compiler/internal/netrt/netrt_unsupported.go"),
		},
		MissingFacts: []string{
			"WASI/web event adapter implementation",
			"browser or WASI readiness integration smoke",
			"WASI/web cancellation/backpressure validation",
		},
		Evidence: ("/home/tetra/Downloads/tetra_ideal_master_plan_20260602.md::" +
			"P18.3; compiler/internal/netrt/netrt_unsupported.go"),
		Boundary: ("WASI/web event adapters remain future adapter work and are " +
			"not claimed as reactor parity"),
	}
}

func nonblockingAcceptReadWriteRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorNonblockingAcceptReadWrite,
		Name:     "Nonblocking accept/read/write",
		Status:   IOReactorImplementedNarrow,
		Platform: "linux",
		RequiredFacts: []string{
			"Accept4 uses SOCK_NONBLOCK and SOCK_CLOEXEC when requested",
			"Read and Write use direct syscall read/write paths",
			"Recv and Send round trips are covered by the Linux netrt test suite",
		},
		Evidence: ("compiler/internal/netrt/netrt_linux.go::Accept; " +
			"compiler/internal/netrt/netrt_linux.go::Read; " +
			"compiler/internal/netrt/netrt_linux.go::Write; " +
			"compiler/internal/netrt/netrt_linux_test.go::" +
			"TestListenTCP4AcceptsNonblockingConnections; " +
			"compiler/internal/netrt/netrt_linux_test.go::" +
			"TestRecvSendRoundTripOnConnectedTCP"),
		Boundary: ("current evidence covers Linux TCP sockets and syscall-style " +
			"I/O helpers; DNS, TLS, UDP, and portable sockets remain " +
			"outside this row"),
	}
}

func readinessPollingRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorReadinessPolling,
		Name:     "Readiness polling",
		Status:   IOReactorImplementedNarrow,
		Platform: "linux",
		RequiredFacts: []string{
			"Poller.Wait exposes readiness events for registered file descriptors",
			("core.net_epoll_wait_one wait-one readiness and " +
				"wait-one-into flag capture are covered by " +
				"TestNetRuntimeEpollReadiness build/run smokes"),
			"wait-one readiness predicates include EPOLLIN, EPOLLOUT, EPOLLERR, and EPOLLHUP",
		},
		Evidence: ("compiler/internal/netrt/netrt_linux.go::Poller.Wait; " +
			"compiler/compiler_suite_test.go::" +
			"testTargetNetworkingEpollReadiness; " +
			"compiler/compiler_suite_test.go::" +
			"TestX86NetworkingEpollReadinessBuildsAndRunsWhenHostSupports" +
			"X86; compiler/compiler_suite_test.go::" +
			"TestX32NetworkingEpollReadinessBuildsAndRunsWhenHostSupports" +
			"X32"),
		Boundary: ("readiness polling is implemented for the current Linux " +
			"native net runtime; it is not a full cross-platform reactor"),
	}
}

func taskWakeupsFromIORow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorTaskWakeupsFromIO,
		Name:     "Task wakeups from I/O readiness",
		Status:   IOReactorImplementedNarrow,
		Platform: "linux",
		RequiredFacts: []string{
			"I/O readiness wakes server work from poller.Wait in webrt.Server.Serve",
			"poller.Wait events dispatch acceptReady, readReady, and flush paths",
			"compiled networking lifecycle composes with core.task_spawn_i32 and task join smokes",
		},
		Evidence: ("compiler/internal/webrt/server.go::Server.Serve; " +
			"compiler/internal/webrt/server.go::acceptReady; " +
			"compiler/internal/webrt/server.go::readReady; " +
			"compiler/internal/webrt/server.go::flush; " +
			"compiler/compiler_suite_test.go::" +
			"TestX86NetworkingLifecycleRuntimeComposesWithTaskScheduler; " +
			"compiler/compiler_suite_test.go::" +
			"TestX32NetworkingLifecycleRuntimeComposesWithTaskScheduler"),
		Boundary: ("this proves current I/O readiness dispatch and scheduler " +
			"composition smoke evidence; it does not wire a production " +
			"per-core task scheduler into every I/O wait"),
	}
}

func timerIntegrationRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorTimerIntegration,
		Name:     "Timer integration",
		Status:   IOReactorEvidenceOnly,
		Platform: "linux",
		RequiredFacts: []string{
			("webrt.Server.Serve uses a 50ms poll timeout so context " +
				"cancellation and timers are not starved by an indefinite " +
				"wait"),
			"P18.2 task runtime evidence covers sleep_ms and wake in deadline order",
			"timer integration evidence is bounded to poll timeouts plus existing task timer smokes",
		},
		Evidence: ("compiler/internal/webrt/server.go::Server.Serve; " +
			"compiler/compiler_suite_test.go::" +
			"TestTaskSleepTimersWakeInDeadlineOrderBuildAndRun; " +
			"compiler/internal/parallelrt/per_core_scheduler.go::" +
			"timersSleepWakeRow"),
		Boundary: ("timer evidence is report-only composition evidence; no new " +
			"timer wheel, deadline queue, or production scheduler " +
			"integration is claimed"),
	}
}

func cancellationRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorCancellation,
		Name:     "Cancellation",
		Status:   IOReactorEvidenceOnly,
		Platform: "linux",
		RequiredFacts: []string{
			"webrt.Server.Serve checks context cancellation before and after Poller.Wait",
			"Server.Close closes the poller and connections so the serve loop can exit",
			("task_group_cancel evidence remains inherited from P18.2 and " +
				"is not promoted to a full structured-concurrency reactor"),
		},
		Evidence: ("compiler/internal/webrt/server.go::Server.Serve; " +
			"compiler/internal/webrt/server.go::Server.Close; " +
			"compiler/internal/parallelrt/per_core_scheduler.go::" +
			"taskGroupCancelRow; compiler/compiler_suite_test.go"),
		Boundary: ("cancellation evidence covers context-driven server shutdown " +
			"and existing task-group cancellation smokes; no full " +
			"structured-concurrency reactor is claimed"),
	}
}

func backpressureRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorBackpressure,
		Name:     "Backpressure",
		Status:   IOReactorEvidenceOnly,
		Platform: "linux",
		RequiredFacts: []string{
			"webrt flush enables EPOLLOUT interest while the output buffer still has pending bytes",
			"the output buffer is drained before keep-alive reuse or close-after-write",
			"pgrt.Pool.Checkout reports ErrPoolExhausted as bounded PostgreSQL pool backpressure",
		},
		Evidence: ("compiler/internal/webrt/server.go::flush; " +
			"compiler/internal/webrt/server.go::updateInterest; " +
			"compiler/internal/pgrt/pool.go::Checkout; " +
			"compiler/internal/pgrt/pool_test.go"),
		Boundary: ("backpressure evidence covers output-buffer interest updates " +
			"and bounded DB pool exhaustion; it does not claim global " +
			"queue pressure propagation across every runtime subsystem"),
	}
}

func reactorReportRowsRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:     IOReactorReportRows,
		Name:   "Reactor report rows",
		Status: IOReactorEvidenceOnly,
		RequiredFacts: []string{
			"tetra.techempower.benchmark.v1 report validation is available for local web benchmark evidence",
			"ValidateReport rejects weak endpoint, command, threshold, skip-db, and identity evidence",
			"P18.3 reactor coverage itself is schema tetra.runtime.io_reactor.v1",
		},
		Evidence: ("tools/validators/techempower/report.go::ValidateReport; " +
			"tools/validators/techempower/report_test.go::" +
			"TestValidateReportAcceptsFullSixEndpointReport; " +
			"compiler/internal/netrt/io_reactor_coverage.go::" +
			"ValidateIOReactorCoverage"),
		Boundary: ("report rows are local evidence contracts and do not imply " +
			"an official TechEmpower result or production web-stack " +
			"readiness"),
	}
}

func httpSmokeRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:     IOReactorHTTPSmoke,
		Name:   "HTTP smoke",
		Status: IOReactorEvidenceOnly,
		RequiredFacts: []string{
			"webrt server smoke covers plaintext responses",
			"webrt server smoke covers json responses",
			"webrt server smoke covers keep-alive pipelining and partial request reads",
		},
		Evidence: ("compiler/internal/webrt/webrt_test.go::" +
			"TestServerPlaintextKeepAliveAndPipelining; " +
			"compiler/internal/webrt/webrt_test.go::" +
			"TestServerJSONEndpointKeepAliveAndPipelining; " +
			"compiler/internal/webrt/webrt_test.go::" +
			"TestServerHandlesPartialRequestRead"),
		Boundary: ("HTTP smoke is local runtime evidence; P19.2 still owns " +
			"production HTTP/JSON stack promotion"),
	}
}

func dbSmokeRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:     IOReactorDBSmoke,
		Name:   "DB smoke",
		Status: IOReactorEvidenceOnly,
		RequiredFacts: []string{
			"PostgreSQL wire helpers are exercised through fake server smoke tests",
			"DB handler smoke proves prepared statement usage",
			"DB handler smoke uses a bounded pgrt pool",
		},
		Evidence: ("compiler/internal/webrt/webrt_test.go::" +
			"TestServerDBEndpointUsesPoolAndSerializesWorld; " +
			"compiler/internal/webrt/webrt_test.go::" +
			"TestServerQueriesEndpointNormalizesCountAndSerializesWorldAr" +
			"ray; compiler/internal/webrt/webrt_test.go::" +
			"TestServerUpdatesEndpointReadsUpdatesThenSerializesWorldArra" +
			"y; compiler/internal/pgrt/pool.go"),
		Boundary: ("DB smoke is local PostgreSQL protocol and handler evidence; " +
			"P19.3 still owns production driver and pool promotion"),
	}
}

func stressEvidenceRow() IOReactorEvidenceRow {
	return IOReactorEvidenceRow{
		ID:       IOReactorStressEvidence,
		Name:     "Stress evidence",
		Status:   IOReactorEvidenceOnly,
		Platform: "linux",
		RequiredFacts: []string{
			("go test -race ./compiler/internal/netrt is the applicable " +
				"race evidence gate for the Go netrt model"),
			"many readiness waits are covered by TestPollerHandlesManyReadinessWaitsAndTimeouts",
			"compiled net runtime server readiness smokes use process timeouts to reject hangs",
		},
		Evidence: ("compiler/internal/netrt/netrt_linux_test.go::" +
			"TestPollerHandlesManyReadinessWaitsAndTimeouts; " +
			"compiler/compiler_suite_test.go::" +
			"runTargetTCPServerReadinessOrSkip; go test -race " +
			"./compiler/internal/netrt"),
		Boundary: ("stress evidence is focused on netrt and compiled readiness " +
			"smokes; it does not prove full cross-target race safety or " +
			"production web throughput"),
	}
}

func requireIOReactorCoverageFacts(row IOReactorEvidenceRow, label string, wants ...string) error {
	for _, want := range wants {
		if !containsIOReactorText(row.RequiredFacts, want) {
			return fmt.Errorf("io reactor coverage: %s row %q missing fact %q", label, row.ID, want)
		}
	}
	return nil
}

func containsIOReactorText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
