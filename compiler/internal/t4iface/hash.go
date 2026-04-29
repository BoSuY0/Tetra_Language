package t4iface

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const HashPrefix = "// t4i-hash: sha256:"

func FingerprintBody(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func HashHeaderForBody(body []byte) string {
	return HashPrefix + strings.TrimPrefix(FingerprintBody(body), "sha256:")
}

func WithHashHeader(body []byte) []byte {
	var out bytes.Buffer
	out.WriteString(HashHeaderForBody(body))
	out.WriteByte('\n')
	out.Write(body)
	return out.Bytes()
}

func SplitHashHeader(raw []byte) (hash string, body []byte, ok bool, err error) {
	line, rest, found := bytes.Cut(raw, []byte{'\n'})
	if !found {
		line = bytes.TrimSuffix(raw, []byte{'\r'})
		rest = nil
	}
	line = bytes.TrimSuffix(line, []byte{'\r'})
	text := string(line)
	if !strings.HasPrefix(text, HashPrefix) {
		return "", raw, false, nil
	}
	hexPart := strings.TrimPrefix(text, HashPrefix)
	if len(hexPart) != 64 {
		return "", nil, true, fmt.Errorf("invalid .t4i hash length")
	}
	if _, decodeErr := hex.DecodeString(hexPart); decodeErr != nil {
		return "", nil, true, fmt.Errorf("invalid .t4i hash: %w", decodeErr)
	}
	return "sha256:" + hexPart, rest, true, nil
}

func ValidateHash(raw []byte) (string, error) {
	hash, body, ok, err := SplitHashHeader(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("missing .t4i hash header")
	}
	actual := FingerprintBody(body)
	if hash != actual {
		return "", fmt.Errorf("invalid .t4i hash: declared %s, computed %s", hash, actual)
	}
	return hash, nil
}
