package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxLSPContentLength = 16 * 1024 * 1024

func readLSPMessage(reader *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid LSP header %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length")
			}
			if parsed > maxLSPContentLength {
				return nil, fmt.Errorf("Content-Length too large: %d exceeds max %d", parsed, maxLSPContentLength)
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeLSPResponse(w io.Writer, id any, result any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeLSPNotification(w io.Writer, method string, params any) error {
	return writeLSPMessage(w, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func writeLSPMessage(w io.Writer, msg any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	raw := bytes.TrimRight(b.Bytes(), "\n")
	_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	return err
}
