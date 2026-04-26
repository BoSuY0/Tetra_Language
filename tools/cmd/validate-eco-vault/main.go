package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type vaultIndex struct {
	RecordsRaw json.RawMessage `json:"records"`
	Records    []vaultRecord
}

type vaultRecord struct {
	Hash   string `json:"hash"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Size   int    `json:"size"`
}

func main() {
	var store string
	flag.StringVar(&store, "store", "", "path to local Eco/Todex vault store")
	flag.Parse()

	if store == "" {
		fmt.Fprintln(os.Stderr, "error: --store is required")
		os.Exit(2)
	}
	if err := validateEcoVault(store); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoVault(store string) error {
	info, err := os.Stat(store)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", store)
	}
	raw, err := os.ReadFile(filepath.Join(store, "records.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing records.json")
		}
		return err
	}
	var index vaultIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return err
	}
	if err := unmarshalRecords(index.RecordsRaw, &index.Records); err != nil {
		return err
	}
	if len(index.Records) == 0 {
		return fmt.Errorf("records must not be empty")
	}
	seen := map[string]bool{}
	for _, record := range index.Records {
		if err := validateVaultRecord(store, record); err != nil {
			return err
		}
		key := record.Hash + "\x00" + record.Kind + "\x00" + record.Source
		if seen[key] {
			return fmt.Errorf("duplicate record %s %s %s", record.Hash, record.Kind, record.Source)
		}
		seen[key] = true
	}
	return nil
}

func unmarshalRecords(raw json.RawMessage, out *[]vaultRecord) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("records must be an array")
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("records must be an array, not null")
	}
	if err := json.Unmarshal(trimmed, out); err != nil {
		return fmt.Errorf("records: %w", err)
	}
	return nil
}

func validateVaultRecord(store string, record vaultRecord) error {
	hexHash, err := parseSHA256Hash(record.Hash)
	if err != nil {
		return err
	}
	if record.Kind == "" {
		return fmt.Errorf("record %s missing kind", record.Hash)
	}
	if !validVaultKind(record.Kind) {
		return fmt.Errorf("record %s has unsupported kind %s", record.Hash, record.Kind)
	}
	if record.Source == "" {
		return fmt.Errorf("record %s missing source", record.Hash)
	}
	if record.Size < 0 {
		return fmt.Errorf("record %s has negative size", record.Hash)
	}
	objectPath := filepath.Join(store, "objects", "sha256", hexHash)
	raw, err := os.ReadFile(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing object %s", objectPath)
		}
		return err
	}
	if len(raw) != record.Size {
		return fmt.Errorf("record %s size mismatch: got %d, object has %d", record.Hash, record.Size, len(raw))
	}
	sum := sha256.Sum256(raw)
	actual := hex.EncodeToString(sum[:])
	if actual != hexHash {
		return fmt.Errorf("record %s hash mismatch: object hashes to sha256:%s", record.Hash, actual)
	}
	return nil
}

func validVaultKind(kind string) bool {
	switch kind {
	case "source", "interface", "build", "test":
		return true
	default:
		return false
	}
}

func parseSHA256Hash(hash string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	hexHash := strings.TrimPrefix(hash, prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	return hexHash, nil
}
