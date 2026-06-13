package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tetra_language/compiler/actorwire"
	"tetra_language/tools/validators/actordist"
)

type smokeOptions struct {
	ReportPath string
	TetraPath  string
	KeepWork   bool
}

type smokeRunner struct {
	opt                  smokeOptions
	workDir              string
	tetraPath            string
	broker               *brokerProcess
	counts               actordist.FrameCounts
	processes            []actordist.ProcessReport
	cases                []actordist.CaseReport
	frameOrder           []string
	nextPeer             uint16
	expectedDecodeErrors int64
}

type brokerProcess struct {
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	done       chan processResult
	reportPath string
	addr       string
}

type processResult struct {
	exitCode int
	output   string
	err      error
}

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.actors.distributed-runtime.v1 report")
	flag.StringVar(&opt.TetraPath, "tetra", "", "tetra CLI path; defaults to a fresh temp build from ./cli/cmd/tetra")
	flag.BoolVar(&opt.KeepWork, "keep-work", false, "keep temporary build directory")
	flag.Parse()
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := runSmoke(context.Background(), opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSmoke(ctx context.Context, opt smokeOptions) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("distributed actor runtime smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	workDir, err := os.MkdirTemp("", "tetra-distributed-actors-*")
	if err != nil {
		return err
	}
	r := &smokeRunner{opt: opt, workDir: workDir, nextPeer: 3}
	if !opt.KeepWork {
		defer os.RemoveAll(workDir)
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	if opt.TetraPath == "" {
		r.tetraPath = filepath.Join(workDir, "tetra")
		if err := runCommand(ctx, "go", "build", "-o", r.tetraPath, "./cli/cmd/tetra"); err != nil {
			return fmt.Errorf("build smoke tetra CLI: %w", err)
		}
	} else {
		r.tetraPath = opt.TetraPath
	}

	if err := r.startBroker(ctx); err != nil {
		return err
	}
	defer r.stopBroker()

	if err := r.runSenderCase(ctx); err != nil {
		return err
	}
	if err := r.runInboundCase(ctx, "cross-node i32 send/receive", "recv_i32", recvI32Source, actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: 2,
		DestNodeID:   1,
		SequenceID:   101,
		ActorID:      0,
		Payload:      []int32{42},
	}); err != nil {
		return err
	}
	if err := r.runInboundCase(ctx, "cross-node tagged send/receive", "recv_tagged", recvTaggedSource, actorwire.Frame{
		Type:         actorwire.FrameSendMsg,
		SourceNodeID: 2,
		DestNodeID:   1,
		SequenceID:   102,
		ActorID:      0,
		Tag:          99,
		Payload:      []int32{8},
	}); err != nil {
		return err
	}
	if err := r.runInboundCase(ctx, "cross-node typed send/receive", "recv_typed", recvTypedSource, actorwire.Frame{
		Type:         actorwire.FrameSendTyped,
		SourceNodeID: 2,
		DestNodeID:   1,
		SequenceID:   103,
		ActorID:      0,
		Tag:          0,
		Payload:      []int32{20, 22},
	}); err != nil {
		return err
	}
	if err := r.runNodeDownCase(ctx); err != nil {
		return err
	}
	if err := r.runTaskCancelJoinCase(ctx); err != nil {
		return err
	}
	if err := r.forceBrokerDroppedFrame(); err != nil {
		return err
	}
	if err := r.runMalformedFrameLengthCase(); err != nil {
		return err
	}
	if err := r.runUnknownFrameTypeCase(); err != nil {
		return err
	}
	if err := r.runBadTypedSlotCountCase(); err != nil {
		return err
	}
	if err := r.runDuplicateNodeCase(); err != nil {
		return err
	}
	if err := r.runForgedSourceNodeCase(); err != nil {
		return err
	}
	brokerAddr := r.broker.addr
	if err := r.stopBroker(); err != nil {
		return err
	}
	if err := r.runMissingNodeAfterBrokerCloseCase(brokerAddr); err != nil {
		return err
	}
	return r.writeReport(ctx)
}

func (r *smokeRunner) startBroker(ctx context.Context) error {
	reportPath := filepath.Join(r.workDir, "actornet-broker.json")
	bctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(bctx, r.tetraPath, "actor-net", "--addr", "127.0.0.1:0", "--report", reportPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start actor-net broker: %w", err)
	}
	done := make(chan processResult, 1)
	go func() {
		err := cmd.Wait()
		done <- processResult{exitCode: processExitCode(err), output: stderr.String(), err: err}
	}()
	addr, err := readBrokerAddr(stdout)
	if err != nil {
		cancel()
		_ = cmd.Process.Kill()
		return err
	}
	r.broker = &brokerProcess{
		cmd:        cmd,
		cancel:     cancel,
		done:       done,
		reportPath: reportPath,
		addr:       addr,
	}
	return nil
}

func readBrokerAddr(stdout io.Reader) (string, error) {
	type result struct {
		addr string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				ch <- result{err: err}
				return
			}
			ch <- result{err: errors.New("actor-net broker exited before printing listen address")}
			return
		}
		line := scanner.Text()
		const prefix = "Actor network broker listening on "
		if !strings.HasPrefix(line, prefix) {
			ch <- result{err: fmt.Errorf("unexpected actor-net broker output %q", line)}
			return
		}
		ch <- result{addr: strings.TrimSpace(strings.TrimPrefix(line, prefix))}
	}()
	select {
	case res := <-ch:
		return res.addr, res.err
	case <-time.After(5 * time.Second):
		return "", errors.New("timed out waiting for actor-net broker listen address")
	}
}

func (r *smokeRunner) stopBroker() error {
	if r.broker == nil {
		return nil
	}
	b := r.broker
	r.broker = nil
	if b.cmd.Process != nil {
		_ = b.cmd.Process.Signal(os.Interrupt)
	}
	select {
	case res := <-b.done:
		b.cancel()
		if res.err != nil && res.exitCode != 0 {
			return fmt.Errorf("actor-net broker exited with %d: %s", res.exitCode, res.output)
		}
		r.processes = append(r.processes, actordist.ProcessReport{
			Name:     "broker",
			Kind:     "broker",
			Path:     r.tetraPath + " actor-net",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		})
		return nil
	case <-time.After(5 * time.Second):
		b.cancel()
		_ = b.cmd.Process.Kill()
		return errors.New("timed out stopping actor-net broker")
	}
}

func (r *smokeRunner) runSenderCase(ctx context.Context) error {
	peer, err := r.connectPeer(2)
	if err != nil {
		return err
	}
	defer peer.Close()
	outPath, err := r.buildNode(ctx, "node_a_sender", senderSource(r.port()))
	if err != nil {
		return err
	}
	cmd, done, err := startNode(ctx, outPath)
	if err != nil {
		return err
	}
	_ = cmd
	seen := map[actorwire.FrameType]bool{}
	deadline := time.Now().Add(3 * time.Second)
	for len(seen) < 4 && time.Now().Before(deadline) {
		frame, err := readFrame(peer)
		if err != nil {
			select {
			case res := <-done:
				return fmt.Errorf("read sender frame after node exit code %d: %w output=%q", res.exitCode, err, res.output)
			default:
			}
			return fmt.Errorf("read sender frame: %w", err)
		}
		r.countFrame(frame.Type)
		seen[frame.Type] = true
		if frame.Type == actorwire.FrameSpawnReq {
			ack := actorwire.Frame{
				Type:         actorwire.FrameSpawnAck,
				SourceNodeID: 2,
				DestNodeID:   1,
				SequenceID:   frame.SequenceID,
				ActorID:      frame.ActorID,
				Status:       actorwire.StatusOK,
			}
			if err := writeFrame(peer, ack); err != nil {
				return err
			}
			r.countFrame(ack.Type)
		}
	}
	for _, typ := range []actorwire.FrameType{actorwire.FrameSpawnReq, actorwire.FrameSendI32, actorwire.FrameSendMsg, actorwire.FrameSendTyped} {
		if !seen[typ] {
			return fmt.Errorf("sender case missing %s frame", frameTypeName(typ))
		}
	}
	res := <-done
	if res.err != nil {
		return fmt.Errorf("node-a sender failed: %w output=%q", res.err, res.output)
	}
	r.recordProcess("node-a-sender", outPath, res)
	return nil
}

func (r *smokeRunner) runInboundCase(ctx context.Context, caseName, nodeName string, source func(int) string, frame actorwire.Frame) error {
	peerID := r.allocPeerNodeID()
	frame.SourceNodeID = peerID
	peer, err := r.connectPeer(peerID)
	if err != nil {
		return err
	}
	defer peer.Close()
	outPath, err := r.buildNode(ctx, nodeName, source(r.port()))
	if err != nil {
		return err
	}
	_, done, err := startNode(ctx, outPath)
	if err != nil {
		return err
	}
	res, err := r.sendUntilNodeExits(peer, done, frame)
	if err != nil {
		return err
	}
	r.countFrame(frame.Type)
	r.recordCase(caseName, res.exitCode, 1)
	r.recordProcess(nodeName, outPath, res)
	return nil
}

func (r *smokeRunner) runNodeDownCase(ctx context.Context) error {
	peerID := r.allocPeerNodeID()
	peer, err := r.connectPeer(peerID)
	if err != nil {
		return err
	}
	defer peer.Close()
	outPath, err := r.buildNode(ctx, "node_down_status", nodeDownSource(r.port()))
	if err != nil {
		return err
	}
	_, done, err := startNode(ctx, outPath)
	if err != nil {
		return err
	}
	frames := []actorwire.Frame{
		{
			Type:         actorwire.FrameSendI32,
			SourceNodeID: peerID,
			DestNodeID:   1,
			SequenceID:   201,
			Payload:      []int32{7},
		},
		{
			Type:         actorwire.FrameNodeDown,
			SourceNodeID: peerID,
			DestNodeID:   1,
			SequenceID:   202,
			Status:       actorwire.StatusNodeUnavailable,
		},
	}
	var res processResult
	deadline := time.Now().Add(3 * time.Second)
	for {
		for _, frame := range frames {
			_ = writeFrame(peer, frame)
		}
		select {
		case res = <-done:
			for _, frame := range frames {
				r.countFrame(frame.Type)
			}
			if res.err != nil {
				return fmt.Errorf("node-down case failed: %w output=%q", res.err, res.output)
			}
			r.recordCase("missing-node failure/status", res.exitCode, 1)
			r.recordProcess("node-down-status", outPath, res)
			return nil
		case <-time.After(50 * time.Millisecond):
			if time.Now().After(deadline) {
				return errors.New("node-down case timed out")
			}
		}
	}
}

func (r *smokeRunner) runTaskCancelJoinCase(ctx context.Context) error {
	outPath, err := r.buildNode(ctx, "task_cancel_join", taskCancelJoinSource)
	if err != nil {
		return err
	}
	res, err := runNode(ctx, outPath, 3*time.Second)
	if err != nil {
		return err
	}
	if res.err != nil {
		return fmt.Errorf("task cancel/join compatibility failed: %w output=%q", res.err, res.output)
	}
	r.recordCase("task cancel/join compatibility", res.exitCode, 1)
	r.recordProcess("task-cancel-join", outPath, res)
	return nil
}

func (r *smokeRunner) forceBrokerDroppedFrame() error {
	peerID := r.allocPeerNodeID()
	peer, err := r.connectPeer(peerID)
	if err != nil {
		return err
	}
	defer peer.Close()
	frame := actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: peerID,
		DestNodeID:   77,
		SequenceID:   301,
		Payload:      []int32{1},
	}
	if err := writeFrame(peer, frame); err != nil {
		return err
	}
	got, err := readFrame(peer)
	if err != nil {
		return err
	}
	if got.Type != actorwire.FrameNodeDown {
		return fmt.Errorf("missing-destination broker response = %s, want node_down", frameTypeName(got.Type))
	}
	r.countFrame(got.Type)
	return nil
}

func (r *smokeRunner) runMalformedFrameLengthCase() error {
	frame := actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: r.allocPeerNodeID(),
		DestNodeID:   1,
		SequenceID:   401,
		Payload:      []int32{1},
	}
	return r.runMalformedRawFrameCase("malformed frame length rejected", frame, func(raw []byte) []byte {
		return raw[:actorwire.FrameSize-1]
	})
}

func (r *smokeRunner) runUnknownFrameTypeCase() error {
	frame := actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: r.allocPeerNodeID(),
		DestNodeID:   1,
		SequenceID:   402,
		Payload:      []int32{1},
	}
	return r.runMalformedRawFrameCase("unknown frame type rejected", frame, func(raw []byte) []byte {
		binary.LittleEndian.PutUint16(raw[actorwire.FrameTypeOffset:], 0xffff)
		return raw
	})
}

func (r *smokeRunner) runBadTypedSlotCountCase() error {
	frame := actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: r.allocPeerNodeID(),
		DestNodeID:   1,
		SequenceID:   403,
		Payload:      []int32{1},
	}
	return r.runMalformedRawFrameCase("bad typed slot count rejected", frame, func(raw []byte) []byte {
		binary.LittleEndian.PutUint16(raw[actorwire.FrameTypeOffset:], uint16(actorwire.FrameSendTyped))
		binary.LittleEndian.PutUint16(raw[actorwire.FrameSlotCountOffset:], actorwire.MaxPayloadSlots+1)
		return raw
	})
}

func (r *smokeRunner) runMalformedRawFrameCase(name string, frame actorwire.Frame, mutate func([]byte) []byte) error {
	raw, err := actorwire.EncodeFrame(frame)
	if err != nil {
		return err
	}
	if err := r.writeRawBrokerFrame(mutate(raw)); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	r.expectedDecodeErrors++
	r.recordNetworkNegativeCase(name)
	return nil
}

func (r *smokeRunner) runDuplicateNodeCase() error {
	peerID := r.allocPeerNodeID()
	primary, err := r.connectPeer(peerID)
	if err != nil {
		return err
	}
	defer primary.Close()

	duplicate, err := net.DialTimeout("tcp", r.broker.addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer duplicate.Close()

	hello := actorwire.Frame{Type: actorwire.FrameHello, SourceNodeID: peerID, DestNodeID: peerID}
	if err := writeFrame(duplicate, hello); err != nil {
		return err
	}
	r.countFrame(hello.Type)
	got, err := readFrame(duplicate)
	if err != nil {
		return err
	}
	r.countFrame(got.Type)
	if got.Type != actorwire.FrameError || got.Status != actorwire.StatusDuplicateNode {
		return fmt.Errorf("duplicate node response = %s status %d, want error duplicate_node", frameTypeName(got.Type), got.Status)
	}
	r.recordNetworkNegativeCase("duplicate node rejected")
	return nil
}

func (r *smokeRunner) runForgedSourceNodeCase() error {
	sourceID := r.allocPeerNodeID()
	destID := r.allocPeerNodeID()
	source, err := r.connectPeer(sourceID)
	if err != nil {
		return err
	}
	defer source.Close()
	dest, err := r.connectPeer(destID)
	if err != nil {
		return err
	}
	defer dest.Close()

	frame := actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: destID,
		DestNodeID:   destID,
		SequenceID:   404,
		Payload:      []int32{99},
	}
	if err := writeFrame(source, frame); err != nil {
		return err
	}
	r.countFrame(frame.Type)
	got, err := readFrame(source)
	if err != nil {
		return err
	}
	r.countFrame(got.Type)
	if got.Type != actorwire.FrameError || got.Status != actorwire.StatusDecodeError {
		return fmt.Errorf("forged source response = %s status %d, want error decode_error", frameTypeName(got.Type), got.Status)
	}
	if err := expectNoFrame(dest, 50*time.Millisecond); err != nil {
		return err
	}
	r.recordNetworkNegativeCase("forged source node rejected")
	return nil
}

func (r *smokeRunner) runMissingNodeAfterBrokerCloseCase(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return errors.New("broker accepted connection after broker close")
	}
	r.recordNetworkNegativeCase("missing-node send after broker close")
	return nil
}

func (r *smokeRunner) writeRawBrokerFrame(data []byte) error {
	conn, err := net.DialTimeout("tcp", r.broker.addr, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (r *smokeRunner) allocPeerNodeID() uint16 {
	id := r.nextPeer
	r.nextPeer++
	if id == 0 || id > actorwire.MaxNodeID {
		id = actorwire.MaxNodeID
	}
	return id
}

func (r *smokeRunner) sendUntilNodeExits(conn net.Conn, done <-chan processResult, frame actorwire.Frame) (processResult, error) {
	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = writeFrame(conn, frame)
		select {
		case res := <-done:
			if res.err != nil {
				return res, fmt.Errorf("node process failed: %w output=%q", res.err, res.output)
			}
			return res, nil
		case <-time.After(50 * time.Millisecond):
			if time.Now().After(deadline) {
				return processResult{}, errors.New("node process timed out")
			}
		}
	}
}

func (r *smokeRunner) connectPeer(nodeID uint16) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", r.broker.addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	hello := actorwire.Frame{Type: actorwire.FrameHello, SourceNodeID: nodeID, DestNodeID: nodeID}
	if err := writeFrame(conn, hello); err != nil {
		_ = conn.Close()
		return nil, err
	}
	r.countFrame(hello.Type)
	ack, err := readFrame(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	r.countFrame(ack.Type)
	if ack.Type != actorwire.FrameHelloAck || ack.Status != actorwire.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("peer hello ack = %s status %d, want hello_ack ok", frameTypeName(ack.Type), ack.Status)
	}
	return conn, nil
}

func (r *smokeRunner) buildNode(ctx context.Context, name string, source string) (string, error) {
	srcPath := filepath.Join(r.workDir, name+".tetra")
	outPath := filepath.Join(r.workDir, name)
	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		return "", err
	}
	if err := runCommand(ctx, r.tetraPath, "build", "--target", "linux-x64", "-o", outPath, srcPath); err != nil {
		return "", fmt.Errorf("build %s: %w", name, err)
	}
	return outPath, nil
}

func (r *smokeRunner) port() int {
	_, portRaw, _ := net.SplitHostPort(r.broker.addr)
	port, _ := strconv.Atoi(portRaw)
	return port
}

func (r *smokeRunner) recordProcess(name string, path string, res processResult) {
	r.processes = append(r.processes, actordist.ProcessReport{
		Name:     name,
		Kind:     "node",
		Path:     path,
		Ran:      true,
		Pass:     res.err == nil && res.exitCode == 0,
		ExitCode: intPtr(res.exitCode),
	})
}

func (r *smokeRunner) recordCase(name string, exitCode int, nodeProcesses int) {
	r.cases = append(r.cases, actordist.CaseReport{
		Name:          name,
		Ran:           true,
		Pass:          exitCode == 0,
		ExpectedExit:  0,
		ActualExit:    intPtr(exitCode),
		NodeProcesses: nodeProcesses,
	})
}

func (r *smokeRunner) recordNetworkNegativeCase(name string) {
	r.cases = append(r.cases, actordist.CaseReport{
		Name:          name,
		Kind:          "network_negative",
		Ran:           true,
		Pass:          true,
		ExpectedExit:  0,
		ActualExit:    intPtr(0),
		NodeProcesses: 0,
	})
}

func (r *smokeRunner) countFrame(typ actorwire.FrameType) {
	switch typ {
	case actorwire.FrameHello:
		r.counts.Hello++
	case actorwire.FrameHelloAck:
		r.counts.HelloAck++
	case actorwire.FrameSpawnReq:
		r.counts.SpawnReq++
	case actorwire.FrameSpawnAck:
		r.counts.SpawnAck++
	case actorwire.FrameSendI32:
		r.counts.SendI32++
	case actorwire.FrameSendMsg:
		r.counts.SendMsg++
	case actorwire.FrameSendTyped:
		r.counts.SendTyped++
	case actorwire.FrameNodeDown:
		r.counts.NodeDown++
	case actorwire.FrameError:
		r.counts.Error++
	}
	r.frameOrder = append(r.frameOrder, frameTypeName(typ))
}

func (r *smokeRunner) writeReport(ctx context.Context) error {
	broker := actordist.BrokerReport{}
	rawBroker, err := os.ReadFile(filepath.Join(r.workDir, "actornet-broker.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawBroker, &broker); err != nil {
		return err
	}
	broker.ExpectedDecodeErrors = r.expectedDecodeErrors
	head, err := currentGitHead(ctx)
	if err != nil {
		return err
	}
	report := actordist.Report{
		Schema:         actordist.SchemaV1,
		Status:         "pass",
		Target:         "linux-x64",
		Host:           "linux-x64",
		Runtime:        "actornet",
		Transport:      "loopback-tcp",
		GitHead:        head,
		ArtifactHashes: "artifact-hashes.json",
		Claims:         []string{"linux-x64 loopback tcp distributed actor runtime evidence"},
		NonClaims: []string{
			"no cluster membership",
			"no reconnect/retry production",
			"no non-linux distributed actor runtime support",
		},
		Broker:      broker,
		Processes:   r.processes,
		FrameCounts: r.counts,
		FrameOrder:  append([]string(nil), r.frameOrder...),
		Cases:       r.cases,
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := actordist.ValidateReport(raw); err != nil {
		return err
	}
	return os.WriteFile(r.opt.ReportPath, append(raw, '\n'), 0o644)
}

func currentGitHead(ctx context.Context) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if cctx.Err() == context.DeadlineExceeded {
		return "", errors.New("git rev-parse HEAD timed out")
	}
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	head := strings.TrimSpace(string(output))
	if len(head) != 40 {
		return "", fmt.Errorf("git rev-parse HEAD returned %q, want 40 hex characters", head)
	}
	return head, nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	output, err := cmd.CombinedOutput()
	if cctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%s timed out", name)
	}
	if err != nil {
		return fmt.Errorf("%s %s: %w output=%q", name, strings.Join(args, " "), err, string(output))
	}
	return nil
}

func startNode(ctx context.Context, path string) (*exec.Cmd, <-chan processResult, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	cmd := exec.CommandContext(cctx, path)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	done := make(chan processResult, 1)
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, nil, err
	}
	go func() {
		err := cmd.Wait()
		cancel()
		done <- processResult{exitCode: processExitCode(err), output: output.String(), err: err}
	}()
	return cmd, done, nil
}

func runNode(ctx context.Context, path string, timeout time.Duration) (processResult, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, path)
	output, err := cmd.CombinedOutput()
	res := processResult{exitCode: processExitCode(err), output: string(output), err: err}
	if cctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("%s timed out", path)
	}
	return res, nil
}

func writeFrame(conn net.Conn, frame actorwire.Frame) error {
	data, err := actorwire.EncodeFrame(frame)
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}

func readFrame(conn net.Conn) (actorwire.Frame, error) {
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return actorwire.Frame{}, err
	}
	data := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(conn, data); err != nil {
		return actorwire.Frame{}, err
	}
	return actorwire.DecodeFrame(data)
}

func expectNoFrame(conn net.Conn, timeout time.Duration) error {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	data := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(conn, data); err == nil {
		return errors.New("unexpected frame received")
	} else if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		return err
	}
	return nil
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return exitErr.ExitCode()
	}
	return 1
}

func intPtr(v int) *int { return &v }

func frameTypeName(typ actorwire.FrameType) string {
	switch typ {
	case actorwire.FrameHello:
		return "hello"
	case actorwire.FrameHelloAck:
		return "hello_ack"
	case actorwire.FrameSpawnReq:
		return "spawn_req"
	case actorwire.FrameSpawnAck:
		return "spawn_ack"
	case actorwire.FrameSendI32:
		return "send_i32"
	case actorwire.FrameSendMsg:
		return "send_msg"
	case actorwire.FrameSendTyped:
		return "send_typed"
	case actorwire.FrameNodeDown:
		return "node_down"
	case actorwire.FrameError:
		return "error"
	default:
		return fmt.Sprintf("frame(%d)", typ)
	}
}

func senderSource(port int) string {
	return fmt.Sprintf(`
enum RemoteMsg:
    case ping(Int)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, %d)
    if connected != 0:
        return 10 + connected
    let peer: actor = core.spawn_remote(2, "worker")
    let sent: Int = core.send(peer, 7)
    if sent != 7:
        return 20 + sent
    let tagged: Int = core.send_msg(peer, 8, 99)
    if tagged != 8:
        return 30 + tagged
    let typed: Int = core.send_typed(peer, RemoteMsg.ping(11))
    if typed != 0:
        return 40 + typed
    return 0
`, port)
}

func recvI32Source(port int) string {
	return fmt.Sprintf(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, %d)
    if connected != 0:
        return 10 + connected
    let msg: Int = core.recv()
    if msg != 42:
        return 80 + msg
    return 0
`, port)
}

func recvTaggedSource(port int) string {
	return fmt.Sprintf(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, %d)
    if connected != 0:
        return 10 + connected
    var raw: actor.msg = core.recv_msg()
    if raw.tag != 99:
        return 20
    if raw.value != 8:
        return 21
    return 0
`, port)
}

func recvTypedSource(port int) string {
	return fmt.Sprintf(`
enum RemoteMsg:
    case ping(Int, Int)
    case reset

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, %d)
    if connected != 0:
        return 10 + connected
    let msg: RemoteMsg = core.recv_typed<RemoteMsg>()
    match msg:
    case RemoteMsg.ping(lhs, rhs):
        if lhs + rhs == 42:
            return 0
        return lhs + rhs
    case RemoteMsg.reset:
        return 90
`, port)
}

func nodeDownSource(port int) string {
	return fmt.Sprintf(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, %d)
    if connected != 0:
        return 10 + connected
    let before: Int = core.actor_node_status(1)
    if before != 0:
        return 20 + before
    let msg: Int = core.recv()
    if msg != 7:
        return 40 + msg
    let _poll: actor.recv_result_i32 = core.recv_poll()
    let after: Int = core.actor_node_status(1)
    if after == 1:
        return 0
    return 30 + after
`, port)
}

const taskCancelJoinSource = `
func worker() -> Int:
    return 77

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error == 1:
        return 0
    return 50 + result.error
`
