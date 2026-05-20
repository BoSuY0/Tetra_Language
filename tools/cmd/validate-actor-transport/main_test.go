package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateActorTransportAcceptsValidReport(t *testing.T) {
	report := validActorTransportReport(t)
	if err := validateActorTransport([]byte(report)); err != nil {
		t.Fatalf("validateActorTransport failed: %v\n%s", err, report)
	}
}

func TestValidateActorTransportRejectsHashMismatch(t *testing.T) {
	report := validActorTransportReportFrom(t, func(report *actorTransportReport) {
		report.MessageSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	})
	if err := validateActorTransport([]byte(report)); err == nil {
		t.Fatalf("expected hash mismatch failure")
	} else if !strings.Contains(err.Error(), "message_sha256 mismatch") {
		t.Fatalf("error = %v, want message_sha256 mismatch", err)
	}
}

func TestValidateActorTransportRejectsMissingReceive(t *testing.T) {
	report := validActorTransportReportFrom(t, func(report *actorTransportReport) {
		report.Trace = report.Trace[:1]
	})
	if err := validateActorTransport([]byte(report)); err == nil {
		t.Fatalf("expected missing receive failure")
	} else if !strings.Contains(err.Error(), "missing receive trace event") {
		t.Fatalf("error = %v, want missing receive trace event", err)
	}
}

func validActorTransportReport(t *testing.T) string {
	t.Helper()
	return validActorTransportReportFrom(t, func(*actorTransportReport) {})
}

func validActorTransportReportFrom(t *testing.T, mutate func(*actorTransportReport)) string {
	t.Helper()
	msg := actorTransportMessage{
		ID:       "msg-1",
		Actor:    "worker",
		Sender:   "main",
		Value:    42,
		Tag:      7,
		Sequence: 1,
	}
	report := actorTransportReport{
		Schema:          actorTransportSchemaV1,
		SourceNode:      "node-a",
		DestinationNode: "node-b",
		Transport:       "tcp-loopback",
		Message:         msg,
		MessageSHA256:   actorTransportMessageSHA256(msg),
		Trace: []actorTransportTraceEvent{
			{Event: "send", Node: "node-a", MessageID: "msg-1"},
			{Event: "receive", Node: "node-b", MessageID: "msg-1"},
		},
	}
	mutate(&report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
