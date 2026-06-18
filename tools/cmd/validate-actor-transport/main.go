package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	actorTransportSchemaV1 = "tetra.actors.transport.v1"
	sha256Prefix           = "sha256:"
)

type actorTransportReport struct {
	Schema          string                     `json:"schema"`
	SourceNode      string                     `json:"source_node"`
	DestinationNode string                     `json:"destination_node"`
	Transport       string                     `json:"transport"`
	Message         actorTransportMessage      `json:"message"`
	MessageSHA256   string                     `json:"message_sha256"`
	Trace           []actorTransportTraceEvent `json:"trace"`
}

type actorTransportMessage struct {
	ID       string `json:"id"`
	Actor    string `json:"actor"`
	Sender   string `json:"sender"`
	Value    int    `json:"value"`
	Tag      int    `json:"tag"`
	Sequence int    `json:"sequence"`
}

type actorTransportTraceEvent struct {
	Event     string `json:"event"`
	Node      string `json:"node"`
	MessageID string `json:"message_id"`
}

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "path to tetra.actors.transport.v1 JSON report")
	flag.Parse()
	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateActorTransport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateActorTransport(raw []byte) error {
	var report actorTransportReport
	if err := decodeStrictActorTransportJSON(raw, &report); err != nil {
		return err
	}
	if report.Schema != actorTransportSchemaV1 {
		return fmt.Errorf("unsupported schema %q", report.Schema)
	}
	if strings.TrimSpace(report.SourceNode) == "" {
		return fmt.Errorf("source_node is required")
	}
	if strings.TrimSpace(report.DestinationNode) == "" {
		return fmt.Errorf("destination_node is required")
	}
	if report.SourceNode == report.DestinationNode {
		return fmt.Errorf("source_node and destination_node must differ")
	}
	switch report.Transport {
	case "tcp-loopback", "tcp":
	default:
		return fmt.Errorf("unsupported transport %q", report.Transport)
	}
	if err := validateActorTransportMessage(report.Message); err != nil {
		return err
	}
	if _, err := parseActorTransportSHA256(report.MessageSHA256); err != nil {
		return fmt.Errorf("invalid message_sha256: %w", err)
	}
	if got := actorTransportMessageSHA256(report.Message); got != report.MessageSHA256 {
		return fmt.Errorf("message_sha256 mismatch: got %s, want %s", report.MessageSHA256, got)
	}
	return validateActorTransportTrace(report)
}

func validateActorTransportMessage(msg actorTransportMessage) error {
	if strings.TrimSpace(msg.ID) == "" {
		return fmt.Errorf("message.id is required")
	}
	if strings.TrimSpace(msg.Actor) == "" {
		return fmt.Errorf("message.actor is required")
	}
	if strings.TrimSpace(msg.Sender) == "" {
		return fmt.Errorf("message.sender is required")
	}
	if msg.Sequence < 0 {
		return fmt.Errorf("message.sequence must not be negative")
	}
	return nil
}

func validateActorTransportTrace(report actorTransportReport) error {
	if len(report.Trace) == 0 {
		return fmt.Errorf("trace must not be empty")
	}
	sendSeen := false
	receiveSeen := false
	for _, event := range report.Trace {
		if event.MessageID != report.Message.ID {
			return fmt.Errorf("trace message_id mismatch: %s", event.MessageID)
		}
		switch event.Event {
		case "send":
			if event.Node != report.SourceNode {
				return fmt.Errorf("send trace node = %s, want %s", event.Node, report.SourceNode)
			}
			sendSeen = true
		case "receive":
			if event.Node != report.DestinationNode {
				return fmt.Errorf(
					"receive trace node = %s, want %s",
					event.Node,
					report.DestinationNode,
				)
			}
			if !sendSeen {
				return fmt.Errorf("receive trace event precedes send")
			}
			receiveSeen = true
		default:
			return fmt.Errorf("unsupported trace event %q", event.Event)
		}
	}
	if !sendSeen {
		return fmt.Errorf("missing send trace event")
	}
	if !receiveSeen {
		return fmt.Errorf("missing receive trace event")
	}
	return nil
}

func actorTransportMessageSHA256(msg actorTransportMessage) string {
	raw, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return sha256Prefix + hex.EncodeToString(sum[:])
}

func parseActorTransportSHA256(hash string) (string, error) {
	if !strings.HasPrefix(hash, sha256Prefix) {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	hexHash := strings.TrimPrefix(hash, sha256Prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	return hexHash, nil
}

func decodeStrictActorTransportJSON(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
