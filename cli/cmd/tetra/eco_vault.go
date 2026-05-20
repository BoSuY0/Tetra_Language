package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type vaultRecord struct {
	Hash   string `json:"hash"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Size   int64  `json:"size"`
}

type vaultIndex struct {
	Records []vaultRecord `json:"records"`
}

func runEcoVault(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco vault <add|list|verify> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco vault <add|list|verify> [options]")
		return 0
	}
	switch args[0] {
	case "add":
		return runEcoVaultAdd(args[1:], stdout, stderr)
	case "list":
		return runEcoVaultList(args[1:], stdout, stderr)
	case "verify":
		return runEcoVaultVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco vault command %q\n", args[0])
		return 2
	}
}

func runEcoVaultAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	kind := fs.String("kind", "source", "record kind: source, interface, build, or test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eco vault add requires one file path")
		return 2
	}
	if !validVaultKind(*kind) {
		fmt.Fprintf(stderr, "unsupported vault kind %q\n", *kind)
		return 2
	}
	record, err := addVaultRecord(*store, fs.Arg(0), *kind)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Vault added: %s %s %s\n", record.Hash, record.Kind, record.Source)
	return 0
}

func runEcoVaultList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault list does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	sortVaultRecords(index.Records)
	for _, record := range index.Records {
		fmt.Fprintf(stdout, "%s %s %s %d\n", record.Hash, record.Kind, record.Source, record.Size)
	}
	return 0
}

func runEcoVaultVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault verify does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, record := range index.Records {
		if err := verifyVaultRecord(*store, record); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Vault OK: %d records\n", len(index.Records))
	return 0
}

func validVaultKind(kind string) bool {
	switch kind {
	case "source", "interface", "build", "test":
		return true
	default:
		return false
	}
}

func addVaultRecord(store string, path string, kind string) (vaultRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return vaultRecord{}, err
	}
	sum := sha256.Sum256(raw)
	hashHex := fmt.Sprintf("%x", sum[:])
	objectPath := vaultObjectPath(store, hashHex)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return vaultRecord{}, err
	}
	if _, err := os.Stat(objectPath); err != nil {
		if !os.IsNotExist(err) {
			return vaultRecord{}, err
		}
		if err := os.WriteFile(objectPath, raw, 0o644); err != nil {
			return vaultRecord{}, err
		}
	}
	record := vaultRecord{
		Hash:   "sha256:" + hashHex,
		Kind:   kind,
		Source: filepath.Clean(path),
		Size:   int64(len(raw)),
	}
	index, err := readVaultIndex(store)
	if err != nil {
		return vaultRecord{}, err
	}
	index.Records = upsertVaultRecord(index.Records, record)
	if err := writeVaultIndex(store, index); err != nil {
		return vaultRecord{}, err
	}
	return record, nil
}

func upsertVaultRecord(records []vaultRecord, record vaultRecord) []vaultRecord {
	for i, existing := range records {
		if existing.Hash == record.Hash && existing.Kind == record.Kind && existing.Source == record.Source {
			records[i] = record
			sortVaultRecords(records)
			return records
		}
	}
	records = append(records, record)
	sortVaultRecords(records)
	return records
}

func readVaultIndex(store string) (vaultIndex, error) {
	path := vaultIndexPath(store)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vaultIndex{}, nil
		}
		return vaultIndex{}, err
	}
	var index vaultIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return vaultIndex{}, err
	}
	sortVaultRecords(index.Records)
	return index, nil
}

func writeVaultIndex(store string, index vaultIndex) error {
	sortVaultRecords(index.Records)
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := vaultIndexPath(store)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func verifyVaultRecord(store string, record vaultRecord) error {
	const prefix = "sha256:"
	if !strings.HasPrefix(record.Hash, prefix) {
		return fmt.Errorf("vault record %s has unsupported hash", record.Source)
	}
	hashHex := strings.TrimPrefix(record.Hash, prefix)
	raw, err := os.ReadFile(vaultObjectPath(store, hashHex))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	actual := fmt.Sprintf("%x", sum[:])
	if actual != hashHex {
		return fmt.Errorf("vault object mismatch for %s", record.Source)
	}
	if int64(len(raw)) != record.Size {
		return fmt.Errorf("vault object size mismatch for %s", record.Source)
	}
	return nil
}

func vaultIndexPath(store string) string {
	return filepath.Join(store, "records.json")
}

func vaultObjectPath(store string, hashHex string) string {
	return filepath.Join(store, "objects", "sha256", hashHex)
}

func sortVaultRecords(records []vaultRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Hash == records[j].Hash {
			if records[i].Kind == records[j].Kind {
				return records[i].Source < records[j].Source
			}
			return records[i].Kind < records[j].Kind
		}
		return records[i].Hash < records[j].Hash
	})
}
