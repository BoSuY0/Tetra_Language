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
	"sort"
	"strings"

	ctarget "tetra_language/compiler/target"
	"tetra_language/tools/internal/reportdecode"
)

const (
	needMapSchemaV1 = "tetra.eco.needmap.v1"
	sha256Prefix    = "sha256:"
)

type needMapReport struct {
	Schema      string          `json:"schema"`
	LockSHA256  string          `json:"lock_sha256,omitempty"`
	CapsulesRaw json.RawMessage `json:"capsules"`
	EdgesRaw    json.RawMessage `json:"edges,omitempty"`
	TargetsRaw  json.RawMessage `json:"targets"`
	Capsules    []needMapNode   `json:"-"`
	Edges       []needMapEdge   `json:"-"`
	Targets     []string        `json:"-"`
}

type needMapNode struct {
	ID                string   `json:"id"`
	Version           string   `json:"version"`
	Targets           []string `json:"targets"`
	Permissions       []string `json:"permissions,omitempty"`
	TransitiveNeedIDs []string `json:"transitive_need_ids,omitempty"`
}

type needMapEdge struct {
	FromID  string `json:"from_id"`
	ToID    string `json:"to_id"`
	Version string `json:"version"`
}

var knownCapsulePermissions = map[string]string{
	"actors":                "actors",
	"alloc":                 "alloc",
	"cap.io":                "io",
	"cap.mem":               "mem",
	"capability":            "capability",
	"control":               "control",
	"fs.read":               "fs.read",
	"fs.readWrite.userData": "fs.readWrite.userData",
	"fs.write":              "fs.write",
	"io":                    "io",
	"io.read":               "io",
	"io.write":              "io",
	"islands":               "islands",
	"link":                  "link",
	"mem":                   "mem",
	"mem.read":              "mem",
	"mem.write":             "mem",
	"mmio":                  "mmio",
	"runtime":               "runtime",
	"runtime.exec":          "runtime",
	"ui":                    "ui",
}

func main() {
	var needMapPath string
	var reportFormat string
	flag.StringVar(&needMapPath, "needmap", "", "path to tetra.eco.needmap.v1 JSON report")
	flag.StringVar(&reportFormat, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if needMapPath == "" {
		fmt.Fprintln(os.Stderr, "error: --needmap is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(needMapPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoNeedMapFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoNeedMap(raw []byte) error {
	return validateEcoNeedMapFormat(raw, "auto")
}

func validateEcoNeedMapFormat(raw []byte, format string) error {
	var report needMapReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return err
	}
	if report.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if report.Schema != needMapSchemaV1 {
		return fmt.Errorf("unsupported needmap schema %q", report.Schema)
	}
	if report.LockSHA256 != "" {
		if _, err := parseSHA256Hash(report.LockSHA256); err != nil {
			return fmt.Errorf("invalid lock_sha256: %w", err)
		}
	}
	if err := unmarshalRequiredArray(report.CapsulesRaw, "capsules", &report.Capsules); err != nil {
		return err
	}
	if len(report.Capsules) == 0 {
		return fmt.Errorf("capsules must not be empty")
	}
	if len(bytes.TrimSpace(report.EdgesRaw)) > 0 {
		if err := unmarshalOptionalArray(report.EdgesRaw, "edges", &report.Edges); err != nil {
			return err
		}
	}
	if err := unmarshalRequiredArray(report.TargetsRaw, "targets", &report.Targets); err != nil {
		return err
	}
	if len(report.Targets) == 0 {
		return fmt.Errorf("targets must not be empty")
	}
	if err := validateNeedMapNodes(report.Capsules); err != nil {
		return err
	}
	if err := validateNeedMapEdges(report.Capsules, report.Edges); err != nil {
		return err
	}
	if err := validateNeedMapTransitiveNeeds(report.Capsules, report.Edges); err != nil {
		return err
	}
	return validateNeedMapTargets(report.Capsules, report.Targets)
}

func unmarshalRequiredArray[T any](raw json.RawMessage, field string, out *[]T) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("%s is required", field)
	}
	return unmarshalOptionalArray(raw, field, out)
}

func unmarshalOptionalArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := decodeStrictJSON(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
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

func validateNeedMapNodes(nodes []needMapNode) error {
	seen := map[string]bool{}
	for _, node := range nodes {
		if node.ID == "" {
			return fmt.Errorf("capsule missing id")
		}
		if !strings.HasPrefix(node.ID, "tetra://") {
			return fmt.Errorf("capsule %s id must use tetra:// prefix", node.ID)
		}
		if node.Version == "" || !isCapsuleSemver(node.Version) {
			return fmt.Errorf("capsule %s version must use semver x.y.z", node.ID)
		}
		if seen[node.ID] {
			return fmt.Errorf("duplicate capsule id %s", node.ID)
		}
		seen[node.ID] = true
		if err := validateTargets("capsule "+node.ID, node.Targets); err != nil {
			return err
		}
		if err := validatePermissions("capsule "+node.ID, node.Permissions); err != nil {
			return err
		}
		if err := validateIDList(
			"capsule "+node.ID+" transitive_need_ids",
			node.ID,
			node.TransitiveNeedIDs,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateNeedMapEdges(nodes []needMapNode, edges []needMapEdge) error {
	byID := nodesByID(nodes)
	seen := map[string]bool{}
	for _, edge := range edges {
		if edge.FromID == "" {
			return fmt.Errorf("edge missing from_id")
		}
		if edge.ToID == "" {
			return fmt.Errorf("edge %s missing to_id", edge.FromID)
		}
		from, ok := byID[edge.FromID]
		if !ok {
			return fmt.Errorf("edge %s -> %s references unknown source", edge.FromID, edge.ToID)
		}
		to, ok := byID[edge.ToID]
		if !ok {
			return fmt.Errorf("edge %s -> %s references unknown target", edge.FromID, edge.ToID)
		}
		if edge.FromID == edge.ToID {
			return fmt.Errorf("edge %s cannot target itself", edge.FromID)
		}
		if edge.Version == "" || !isCapsuleSemver(edge.Version) {
			return fmt.Errorf("edge %s -> %s version must use semver x.y.z", edge.FromID, edge.ToID)
		}
		if edge.Version != to.Version {
			return fmt.Errorf(
				"edge %s -> %s version mismatch: edge has %s, capsule has %s",
				edge.FromID,
				edge.ToID,
				edge.Version,
				to.Version,
			)
		}
		key := from.ID + "\x00" + to.ID + "\x00" + edge.Version
		if seen[key] {
			return fmt.Errorf("duplicate edge %s -> %s", edge.FromID, edge.ToID)
		}
		seen[key] = true
	}
	return nil
}

func validateNeedMapTransitiveNeeds(nodes []needMapNode, edges []needMapEdge) error {
	byID := nodesByID(nodes)
	adjacent := map[string][]string{}
	for _, edge := range edges {
		adjacent[edge.FromID] = append(adjacent[edge.FromID], edge.ToID)
	}
	for _, node := range nodes {
		for _, id := range node.TransitiveNeedIDs {
			if _, ok := byID[id]; !ok {
				return fmt.Errorf(
					"capsule %s transitive_need_ids references unknown capsule %s",
					node.ID,
					id,
				)
			}
		}
		expected := collectTransitiveNeeds(node.ID, adjacent, map[string]bool{})
		sort.Strings(expected)
		actual := sortedStringCopy(node.TransitiveNeedIDs)
		if !sameStringSlice(actual, expected) {
			return fmt.Errorf(
				"capsule %s transitive_need_ids mismatch: has [%s], expected [%s]",
				node.ID,
				strings.Join(actual, ","),
				strings.Join(expected, ","),
			)
		}
	}
	return nil
}

func validateNeedMapTargets(nodes []needMapNode, targets []string) error {
	if err := validateTargets("needmap", targets); err != nil {
		return err
	}
	targetSet := map[string]bool{}
	for _, node := range nodes {
		for _, target := range node.Targets {
			targetSet[target] = true
		}
	}
	expected := make([]string, 0, len(targetSet))
	for target := range targetSet {
		expected = append(expected, target)
	}
	sort.Strings(expected)
	actual := sortedStringCopy(targets)
	if !sameStringSlice(actual, expected) {
		return fmt.Errorf(
			"targets mismatch: has [%s], expected [%s]",
			strings.Join(actual, ","),
			strings.Join(expected, ","),
		)
	}
	return nil
}

func validateTargets(label string, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("%s missing targets", label)
	}
	supported := supportedTargets()
	seen := map[string]bool{}
	for _, target := range targets {
		if target == "" {
			return fmt.Errorf("%s has empty target", label)
		}
		if !supported[target] {
			return fmt.Errorf("%s has unsupported target %s", label, target)
		}
		if seen[target] {
			return fmt.Errorf("%s has duplicate target %s", label, target)
		}
		seen[target] = true
	}
	return nil
}

func validatePermissions(label string, permissions []string) error {
	seen := map[string]bool{}
	for _, permission := range permissions {
		normalized, ok := knownCapsulePermissions[permission]
		if !ok {
			return fmt.Errorf("%s has unknown permission %s", label, permission)
		}
		if seen[normalized] {
			return fmt.Errorf("%s has duplicate permission %s", label, normalized)
		}
		seen[normalized] = true
	}
	return nil
}

func validateIDList(label string, selfID string, ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("%s has empty id", label)
		}
		if !strings.HasPrefix(id, "tetra://") {
			return fmt.Errorf("%s %s must use tetra:// prefix", label, id)
		}
		if id == selfID {
			return fmt.Errorf("%s cannot include itself", label)
		}
		if seen[id] {
			return fmt.Errorf("%s has duplicate id %s", label, id)
		}
		seen[id] = true
	}
	return nil
}

func collectTransitiveNeeds(
	id string,
	adjacent map[string][]string,
	seen map[string]bool,
) []string {
	var out []string
	for _, depID := range adjacent[id] {
		if seen[depID] {
			continue
		}
		seen[depID] = true
		out = append(out, depID)
		out = append(out, collectTransitiveNeeds(depID, adjacent, seen)...)
	}
	return dedupeStrings(out)
}

func nodesByID(nodes []needMapNode) map[string]needMapNode {
	byID := map[string]needMapNode{}
	for _, node := range nodes {
		byID[node.ID] = node
	}
	return byID
}

func supportedTargets() map[string]bool {
	out := map[string]bool{}
	for _, triple := range ctarget.SupportedTriples() {
		out[triple] = true
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		out[triple] = true
	}
	return out
}

func parseSHA256Hash(hash string) (string, error) {
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

func isCapsuleSemver(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func sortedStringCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sameStringSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
