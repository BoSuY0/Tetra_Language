package actorsystem

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SchemaV1 = "tetra.actor.system_messages.v1"
const LayoutSchemaV1 = "tetra.actor.system_layout.v1"
const ArtifactHashSchema = "tetra.release-artifact-hashes.v1alpha1"

const testInjectorSymbol = "__tetra_test_actor_system_inject"

type Options struct {
	CurrentGitHead  string
	RequireCleanGit bool
}

type Report struct {
	Schema            string            `json:"schema"`
	Pass              bool              `json:"pass"`
	Target            string            `json:"target"`
	Host              string            `json:"host"`
	Runtime           string            `json:"runtime"`
	GitHead           string            `json:"git_head"`
	GitDirty          bool              `json:"git_dirty"`
	Design            string            `json:"design"`
	Producer          string            `json:"producer"`
	ReportDir         string            `json:"report_dir"`
	ArtifactHashes    string            `json:"artifact_hashes"`
	CommandLine       string            `json:"command_line"`
	Claims            []string          `json:"claims,omitempty"`
	NonClaims         []string          `json:"nonclaims"`
	API               APICoverage       `json:"api"`
	Isolation         IsolationReport   `json:"isolation"`
	Security          SecurityReport    `json:"security"`
	Events            EventsReport      `json:"events"`
	Memory            MemoryReport      `json:"memory"`
	ReleaseSymbolScan ReleaseSymbolScan `json:"release_symbol_scan"`
	Commands          []CommandReport   `json:"commands"`
	Artifacts         []ArtifactReport  `json:"artifacts"`
}

type APICoverage struct {
	RecvSystem      bool `json:"recv_system"`
	PollSystem      bool `json:"poll_system"`
	RecvSystemUntil bool `json:"recv_system_until"`
}

type IsolationReport struct {
	SeparateHeadsTails         bool `json:"separate_heads_tails"`
	UserRecvSystemConsumptions int  `json:"user_recv_system_consumptions"`
	SystemRecvUserConsumptions int  `json:"system_recv_user_consumptions"`
	UserQueueFIFOViolations    int  `json:"user_queue_fifo_violations"`
	SystemQueueFIFOViolations  int  `json:"system_queue_fifo_violations"`
	SenderUnchanged            bool `json:"sender_unchanged"`
}

type SecurityReport struct {
	OrdinarySendForgeryRejected bool `json:"ordinary_send_forgery_rejected"`
	RuntimeHandlesOpaque        bool `json:"runtime_handles_opaque"`
	ReleaseTestInjectorExported bool `json:"release_test_injector_exported"`
}

type EventsReport struct {
	Exit            int    `json:"exit"`
	Down            int    `json:"down"`
	NodeDownFixture int    `json:"node_down_fixture"`
	DuplicateDown   int    `json:"duplicate_down"`
	Producer        string `json:"producer"`
}

type MemoryReport struct {
	Bounded                bool `json:"bounded"`
	ReservedCredits        int  `json:"reserved_credits"`
	LiveBytesAfterShutdown int  `json:"live_bytes_after_shutdown"`
	SilentDrops            int  `json:"silent_drops"`
}

type ReleaseSymbolScan struct {
	Scanned              bool     `json:"scanned"`
	Binary               string   `json:"binary"`
	ForbiddenSymbols     []string `json:"forbidden_symbols"`
	ExportedSymbols      []string `json:"exported_symbols,omitempty"`
	TestInjectorExported bool     `json:"test_injector_exported"`
}

type CommandReport struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Log     string `json:"log"`
}

type ArtifactReport struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema,omitempty"`
}

type hashManifest struct {
	Schema    string         `json:"schema"`
	Root      string         `json:"root"`
	Artifacts []hashArtifact `json:"artifacts"`
}

type hashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type layoutReport struct {
	Schema      string            `json:"schema"`
	Target      string            `json:"target"`
	Runtime     string            `json:"runtime"`
	Actor       layoutSection     `json:"actor"`
	Scheduler   layoutSection     `json:"scheduler"`
	SystemEvent layoutSection     `json:"system_event"`
	RawTypes    []rawTypeLayout   `json:"raw_types"`
	Invariants  []layoutInvariant `json:"invariants"`
}

type layoutSection struct {
	Name      string        `json:"name"`
	Size      int           `json:"size"`
	Alignment int           `json:"alignment"`
	Fields    []layoutField `json:"fields"`
}

type layoutField struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Size   int    `json:"size"`
	End    int    `json:"end"`
}

type rawTypeLayout struct {
	Name              string `json:"name"`
	Slots             int    `json:"slots"`
	RuntimeOwned      bool   `json:"runtime_owned"`
	UserConstructible bool   `json:"user_constructible"`
}

type layoutInvariant struct {
	Name string `json:"name"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	return ValidateReportWithOptions(raw, Options{})
}

func ValidateReportWithOptions(raw []byte, opts Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if !report.Pass {
		issues = append(issues, "pass is false, want true")
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if report.Host != "linux-x64" {
		issues = append(issues, fmt.Sprintf("host is %q, want linux-x64", report.Host))
	}
	if report.Runtime != "builtin-actor-runtime-v2" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want builtin-actor-runtime-v2", report.Runtime))
	}
	if !isHexGitHead(report.GitHead) {
		issues = append(issues, fmt.Sprintf("git_head is %q, want 40 lowercase hex characters", report.GitHead))
	}
	if opts.CurrentGitHead != "" && report.GitHead != opts.CurrentGitHead {
		issues = append(issues, fmt.Sprintf("git_head %q does not match current git head %q", report.GitHead, opts.CurrentGitHead))
	}
	if opts.RequireCleanGit && report.GitDirty {
		issues = append(issues, "git_dirty is true in clean/final validation mode")
	}
	if report.Design != "separate-system-lane-v1" {
		issues = append(issues, fmt.Sprintf("design is %q, want separate-system-lane-v1", report.Design))
	}
	if report.Producer != "test_hook" {
		issues = append(issues, fmt.Sprintf("producer is %q, want test_hook for P01 fixture evidence", report.Producer))
	}
	if report.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("report_dir is %q, want .", report.ReportDir))
	}
	if report.ArtifactHashes != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("artifact_hashes is %q, want artifact-hashes.json", report.ArtifactHashes))
	}
	if strings.TrimSpace(report.CommandLine) == "" {
		issues = append(issues, "command_line is required")
	}
	issues = append(issues, validateClaims(report.Claims)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateAPI(report.API)...)
	issues = append(issues, validateIsolation(report.Isolation)...)
	issues = append(issues, validateSecurity(report.Security)...)
	issues = append(issues, validateEvents(report.Events)...)
	issues = append(issues, validateMemory(report.Memory)...)
	issues = append(issues, validateReleaseSymbolScan(report.ReleaseSymbolScan)...)
	issues = append(issues, validateCommands(report.Commands)...)
	issues = append(issues, validateArtifacts(report.Artifacts)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func ValidateReportDir(dir string, opts Options) error {
	reportPath := filepath.Join(dir, "actor-system-messages-linux-x64.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(reportPath), err)
	}
	if err := ValidateReportWithOptions(raw, opts); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(reportPath), err)
	}
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	expected := expectedHashSchemas(report.Artifacts)
	delete(expected, "artifact-hashes.json")
	if err := validateArtifactHashManifest(dir, "artifact-hashes.json", expected); err != nil {
		return err
	}
	if err := validateLayoutReport(filepath.Join(dir, "actor-system-layout-linux-x64.json")); err != nil {
		return err
	}
	return nil
}

func validateClaims(claims []string) []string {
	required := "source-level system-message API and isolated runtime system lane implemented for Linux-x64 builtin runtime"
	var hasRequired bool
	var issues []string
	for _, claim := range claims {
		if claim == required {
			hasRequired = true
		}
		lower := strings.ToLower(claim)
		for _, forbidden := range []string{
			"production supervision",
			"supervision tree",
			"authenticated node-down",
			"node-down producer is production",
			"p06 complete",
			"p10 complete",
			"cluster membership",
			"reconnect",
			"retry",
			"tls",
			"mtls",
			"erlang/otp",
			"full actor runtime",
		} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("forbidden P01 actor system-message claim %q mentions %q", claim, forbidden))
			}
		}
	}
	if !hasRequired {
		issues = append(issues, fmt.Sprintf("missing scoped claim %q", required))
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	joined := strings.ToLower(strings.Join(nonclaims, "\n"))
	var issues []string
	for _, required := range []string{
		"p06",
		"p10",
		"erlang/otp",
		"cluster membership",
		"non-linux",
		"zero-copy",
		"formal race",
	} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q boundary", required))
		}
	}
	return issues
}

func validateAPI(api APICoverage) []string {
	var issues []string
	if !api.RecvSystem {
		issues = append(issues, "api.recv_system must be true")
	}
	if !api.PollSystem {
		issues = append(issues, "api.poll_system must be true")
	}
	if !api.RecvSystemUntil {
		issues = append(issues, "api.recv_system_until must be true")
	}
	return issues
}

func validateIsolation(isolation IsolationReport) []string {
	var issues []string
	if !isolation.SeparateHeadsTails {
		issues = append(issues, "isolation.separate_heads_tails must be true")
	}
	if isolation.UserRecvSystemConsumptions != 0 {
		issues = append(issues, "user_recv_system_consumptions must be 0")
	}
	if isolation.SystemRecvUserConsumptions != 0 {
		issues = append(issues, "system_recv_user_consumptions must be 0")
	}
	if isolation.UserQueueFIFOViolations != 0 {
		issues = append(issues, "user_queue_fifo_violations must be 0")
	}
	if isolation.SystemQueueFIFOViolations != 0 {
		issues = append(issues, "system_queue_fifo_violations must be 0")
	}
	if !isolation.SenderUnchanged {
		issues = append(issues, "sender_unchanged must be true")
	}
	return issues
}

func validateSecurity(security SecurityReport) []string {
	var issues []string
	if !security.OrdinarySendForgeryRejected {
		issues = append(issues, "ordinary_send_forgery_rejected must be true")
	}
	if !security.RuntimeHandlesOpaque {
		issues = append(issues, "runtime_handles_opaque must be true")
	}
	if security.ReleaseTestInjectorExported {
		issues = append(issues, testInjectorSymbol+" must not be exported")
	}
	return issues
}

func validateEvents(events EventsReport) []string {
	var issues []string
	if events.Exit < 1 {
		issues = append(issues, "events.exit must be at least 1")
	}
	if events.Down < 1 {
		issues = append(issues, "events.down must be at least 1")
	}
	if events.NodeDownFixture < 1 {
		issues = append(issues, "events.node_down_fixture must be at least 1 for P01")
	}
	if events.DuplicateDown != 0 {
		issues = append(issues, "events.duplicate_down must be 0")
	}
	if events.Producer != "test_hook" {
		issues = append(issues, fmt.Sprintf("events.producer is %q, want test_hook", events.Producer))
	}
	return issues
}

func validateMemory(memory MemoryReport) []string {
	var issues []string
	if !memory.Bounded {
		issues = append(issues, "memory.bounded must be true")
	}
	if memory.ReservedCredits != 0 {
		issues = append(issues, "memory.reserved_credits must be 0")
	}
	if memory.LiveBytesAfterShutdown != 0 {
		issues = append(issues, "memory.live_bytes_after_shutdown must be 0")
	}
	if memory.SilentDrops != 0 {
		issues = append(issues, "memory.silent_drops must be 0")
	}
	return issues
}

func validateReleaseSymbolScan(scan ReleaseSymbolScan) []string {
	var issues []string
	if !scan.Scanned {
		issues = append(issues, "release_symbol_scan.scanned must be true")
	}
	if strings.TrimSpace(scan.Binary) == "" {
		issues = append(issues, "release_symbol_scan.binary is required")
	}
	if !stringInSet(testInjectorSymbol, scan.ForbiddenSymbols) {
		issues = append(issues, "release_symbol_scan.forbidden_symbols must include "+testInjectorSymbol)
	}
	if scan.TestInjectorExported || stringInSet(testInjectorSymbol, scan.ExportedSymbols) {
		issues = append(issues, testInjectorSymbol+" must not be exported")
	}
	return issues
}

func validateCommands(commands []CommandReport) []string {
	required := map[string]bool{
		"focused-validator-tests":        false,
		"actor-system-message-validator": false,
		"generated-examples-build-run":   false,
		"negative-forgery-check":         false,
		"actor-system-layout-report":     false,
		"release-symbol-scan":            false,
		"artifact-hashes-write":          false,
		"artifact-hashes-validate":       false,
	}
	var issues []string
	seen := map[string]bool{}
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			issues = append(issues, "command name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate command %s", name))
		}
		seen[name] = true
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if command.Status != "pass" {
			issues = append(issues, fmt.Sprintf("command %s status is %q, want pass", name, command.Status))
		}
		if strings.TrimSpace(command.Command) == "" {
			issues = append(issues, fmt.Sprintf("command %s command text is required", name))
		}
		if strings.TrimSpace(command.Log) == "" {
			issues = append(issues, fmt.Sprintf("command %s log is required", name))
		}
		for _, forbidden := range []string{"|| true", "continue-on-error", "set +e"} {
			if strings.Contains(command.Command, forbidden) {
				issues = append(issues, fmt.Sprintf("command %s contains bypass marker %q", name, forbidden))
			}
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required command %s", name))
		}
	}
	return issues
}

func validateArtifacts(artifacts []ArtifactReport) []string {
	required := map[string]string{
		"actor-system-messages-linux-x64.json": SchemaV1,
		"actor-system-layout-linux-x64.json":   LayoutSchemaV1,
		"artifact-hashes.json":                 ArtifactHashSchema,
		"bin/system_user_queue_isolation":      "",
	}
	var issues []string
	seen := map[string]bool{}
	for _, artifact := range artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			issues = append(issues, "artifact path is required")
			continue
		}
		if seen[path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact %s", path))
		}
		seen[path] = true
		if filepath.IsAbs(path) || strings.Contains(path, "..") || strings.Contains(path, "\\") {
			issues = append(issues, fmt.Sprintf("artifact path %q must be relative slash path", path))
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("artifact %s kind is required", path))
		}
		if want, ok := required[path]; ok {
			if want != "" && artifact.Schema != want {
				issues = append(issues, fmt.Sprintf("artifact %s schema is %q, want %q", path, artifact.Schema, want))
			}
			delete(required, path)
		}
	}
	for path := range required {
		issues = append(issues, fmt.Sprintf("missing required artifact %s", path))
	}
	return issues
}

func validateLayoutReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	var report layoutReport
	if err := decodeStrict(raw, &report); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	var issues []string
	if report.Schema != LayoutSchemaV1 {
		issues = append(issues, fmt.Sprintf("layout schema is %q, want %q", report.Schema, LayoutSchemaV1))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("layout target is %q, want linux-x64", report.Target))
	}
	if report.Runtime != "builtin-actor-runtime-v2" {
		issues = append(issues, fmt.Sprintf("layout runtime is %q, want builtin-actor-runtime-v2", report.Runtime))
	}
	if report.Actor.Size < 512 {
		issues = append(issues, fmt.Sprintf("actor layout size is %d, want at least 512", report.Actor.Size))
	}
	for _, required := range []string{
		"system_mailbox_head",
		"system_mailbox_tail",
		"system_recv_scratch",
		"wait_kind",
	} {
		if !layoutHasField(report.Actor.Fields, required) {
			issues = append(issues, fmt.Sprintf("actor layout missing %s", required))
		}
	}
	for _, required := range []string{
		"system_event_base",
		"system_event_live_bytes",
		"system_event_reserved_credits",
		"runtime_closing",
	} {
		if !layoutHasField(report.Scheduler.Fields, required) {
			issues = append(issues, fmt.Sprintf("scheduler layout missing %s", required))
		}
	}
	if report.SystemEvent.Size != 64 {
		issues = append(issues, fmt.Sprintf("system_event size is %d, want 64", report.SystemEvent.Size))
	}
	if !layoutHasField(report.SystemEvent.Fields, "node_epoch") {
		issues = append(issues, "system_event layout missing node_epoch")
	}
	for _, required := range []struct {
		name  string
		slots int
	}{
		{name: "actor.node", slots: 2},
		{name: "actor.system_recv_raw", slots: 8},
	} {
		if !layoutHasRuntimeOwnedRawType(report.RawTypes, required.name, required.slots) {
			issues = append(issues, fmt.Sprintf("layout raw_types missing %s/%d", required.name, required.slots))
		}
	}
	for _, invariant := range []string{
		"actor_system_mailbox_within_actor",
		"scheduler_system_event_pool_fields_ordered",
		"system_event_layout_separate_from_user_message",
	} {
		if !layoutHasPassingInvariant(report.Invariants, invariant) {
			issues = append(issues, fmt.Sprintf("layout invariant %s did not pass", invariant))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func layoutHasField(fields []layoutField, name string) bool {
	for _, field := range fields {
		if field.Name == name && field.Offset >= 0 && field.Size > 0 && field.End == field.Offset+field.Size {
			return true
		}
	}
	return false
}

func layoutHasRuntimeOwnedRawType(types []rawTypeLayout, name string, slots int) bool {
	for _, typ := range types {
		if typ.Name == name && typ.Slots == slots && typ.RuntimeOwned && !typ.UserConstructible {
			return true
		}
	}
	return false
}

func layoutHasPassingInvariant(invariants []layoutInvariant, name string) bool {
	for _, invariant := range invariants {
		if invariant.Name == name && invariant.Pass {
			return true
		}
	}
	return false
}

func expectedHashSchemas(artifacts []ArtifactReport) map[string]string {
	expected := map[string]string{}
	for _, artifact := range artifacts {
		expected[artifact.Path] = artifact.Schema
	}
	return expected
}

func validateArtifactHashManifest(root, rel string, expected map[string]string) error {
	manifestPath := filepath.Join(root, rel)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(manifestPath), err)
	}
	var manifest hashManifest
	if err := decodeStrict(raw, &manifest); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(manifestPath), err)
	}
	var issues []string
	if manifest.Schema != ArtifactHashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", rel, manifest.Schema, ArtifactHashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("%s root is %q, want .", rel, manifest.Root))
	}
	seen := map[string]bool{}
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == rel {
			issues = append(issues, fmt.Sprintf("%s must not list itself", rel))
		}
		if filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "..") || strings.Contains(artifact.Path, "\\") {
			issues = append(issues, fmt.Sprintf("%s artifact path %q must be relative slash path", rel, artifact.Path))
			continue
		}
		if seen[artifact.Path] {
			issues = append(issues, fmt.Sprintf("%s duplicate artifact %s", rel, artifact.Path))
		}
		seen[artifact.Path] = true
		if want, ok := expected[artifact.Path]; ok {
			if want != "" && artifact.Schema != want {
				issues = append(issues, fmt.Sprintf("%s artifact %s schema is %q, want %q", rel, artifact.Path, artifact.Schema, want))
			}
			delete(expected, artifact.Path)
		}
		if err := validateHashedArtifact(root, artifact); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for path := range expected {
		issues = append(issues, fmt.Sprintf("%s missing artifact %s", rel, path))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateHashedArtifact(root string, artifact hashArtifact) error {
	path := filepath.Join(root, filepath.FromSlash(artifact.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("artifact-hashes.json artifact %s: %w", artifact.Path, err)
	}
	sum := sha256.Sum256(raw)
	wantSHA := fmt.Sprintf("sha256:%x", sum)
	var issues []string
	if artifact.SHA256 != wantSHA {
		issues = append(issues, fmt.Sprintf("artifact-hashes.json sha256 mismatch for %s: got %s want %s", artifact.Path, artifact.SHA256, wantSHA))
	}
	if artifact.Size != int64(len(raw)) {
		issues = append(issues, fmt.Sprintf("artifact-hashes.json size mismatch for %s: got %d want %d", artifact.Path, artifact.Size, len(raw)))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func isHexGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			if ch < 'a' || ch > 'f' {
				return false
			}
		}
	}
	return true
}

func stringInSet(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func decodeStrict(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple JSON values")
	}
	return nil
}
