package main

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

func TestReadLSPMessageRejectsTooLargeContentLength(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n", maxLSPContentLength+1)))

	body, err := readLSPMessage(reader)
	if err == nil {
		t.Fatalf("readLSPMessage err = nil, body length = %d", len(body))
	}
	if !strings.Contains(err.Error(), "Content-Length too large") {
		t.Fatalf("readLSPMessage err = %q, want Content-Length too large", err.Error())
	}
}

func TestReadLSPMessageReadsNormalContentLength(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("Content-Length: 15\r\n\r\n{\"jsonrpc\":\"2\"}"))

	body, err := readLSPMessage(reader)
	if err != nil {
		t.Fatalf("readLSPMessage err = %v", err)
	}
	if got, want := string(body), `{"jsonrpc":"2"}`; got != want {
		t.Fatalf("readLSPMessage body = %q, want %q", got, want)
	}
}
