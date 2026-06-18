package postv04prod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/memoryprod"
	"tetra_language/tools/validators/nativeui"
	"tetra_language/tools/validators/parallelprod"
	"tetra_language/tools/validators/uiprod"
)

const SchemaV1 = "tetra.release.post_v0_4.memory_parallel_ui_completion_audit.v1"
const DefaultAuditFilename = "post-v0.4-production-audit.json"

type Report struct {
	Schema       string          `json:"schema"`
	Status       string          `json:"status"`
	Target       string          `json:"target"`
	CombinedGate string          `json:"combined_gate"`
	ReportDir    string          `json:"report_dir"`
	Layers       []LayerReport   `json:"layers"`
	Checklist    []ChecklistItem `json:"checklist"`
}

type LayerReport struct {
	Name         string `json:"name"`
	Artifact     string `json:"artifact"`
	Schema       string `json:"schema"`
	Validator    string `json:"validator"`
	Status       string `json:"status"`
	ProcessCount int    `json:"process_count"`
	CaseCount    int    `json:"case_count"`
	AuditCount   int    `json:"audit_count"`
}

type ChecklistItem struct {
	Layer       string `json:"layer"`
	Requirement string `json:"requirement"`
	Artifact    string `json:"artifact"`
	Evidence    string `json:"evidence"`
	Result      string `json:"result"`
}

type RequiredItem struct {
	Layer       string
	Requirement string
}

func RequiredChecklist() []RequiredItem {
	return []RequiredItem{
		{Layer: "memory", Requirement: "stable allocator/runtime memory model"},
		{Layer: "memory", Requirement: "ownership/borrow/consume escape model"},
		{Layer: "memory", Requirement: "heap, slices, structs, and closures memory coverage"},
		{Layer: "memory", Requirement: "unsafe/cap.mem/raw memory/memcpy/memset rules"},
		{Layer: "memory", Requirement: "runtime bounds checks and diagnostics"},
		{Layer: "memory", Requirement: "raw pointer bounds metadata"},
		{Layer: "memory", Requirement: "stress/fuzz evidence"},
		{Layer: "memory", Requirement: "measured memory benchmark improvement"},
		{Layer: "memory", Requirement: "allocator benchmark evidence classification"},
		{
			Layer:       "memory",
			Requirement: "use-after-free, double-free, borrow escape, and aliasing safety",
		},
		{Layer: "memory", Requirement: "actor/task transfer safety"},
		{Layer: "memory", Requirement: "leak/resource finalization evidence"},
		{Layer: "memory", Requirement: "real memory examples"},
		{Layer: "memory", Requirement: "safe memory documentation"},
		{Layer: "memory", Requirement: "release-gate entrypoint"},
		{Layer: "parallelism", Requirement: "production task scheduler"},
		{Layer: "parallelism", Requirement: "join/cancel/deadline/select/group lifecycle"},
		{Layer: "parallelism", Requirement: "actor mailbox backpressure and failure handling"},
		{Layer: "parallelism", Requirement: "task/actor/thread-boundary transfer rules"},
		{Layer: "parallelism", Requirement: "race-safety model or conservative rejections"},
		{
			Layer:       "parallelism",
			Requirement: "stress evidence for tasks, actor messages, cancellation storms, and timeouts",
		},
		{Layer: "parallelism", Requirement: "safe/unsafe/forbidden parallelism documentation"},
		{Layer: "parallelism", Requirement: "stable parallel diagnostics"},
		{Layer: "parallelism", Requirement: "actor benchmark Tier 0/Tier 1 preparation"},
		{Layer: "parallelism", Requirement: "release-gate entrypoint"},
		{Layer: "ui", Requirement: "Linux-x64 desktop UI runtime"},
		{Layer: "ui", Requirement: "window lifecycle"},
		{Layer: "ui", Requirement: "layout system"},
		{Layer: "ui", Requirement: "buttons/text/input/lists/panels widgets"},
		{Layer: "ui", Requirement: "state binding"},
		{Layer: "ui", Requirement: "event loop and redraw/update model"},
		{Layer: "ui", Requirement: "async commands and timers"},
		{Layer: "ui", Requirement: "error/crash handling"},
		{Layer: "ui", Requirement: "real examples and dogfood applications"},
		{Layer: "ui", Requirement: "compiler-emitted UI bundle/native-shell trace load evidence"},
		{Layer: "ui", Requirement: "sidecar-driven native UI runtime integration"},
		{Layer: "ui", Requirement: "stable UI diagnostics"},
		{Layer: "ui", Requirement: "release-gate entrypoint rejecting runtime-less evidence"},
		{Layer: "combined", Requirement: "ordered Memory Parallelism UI gate"},
		{Layer: "combined", Requirement: "artifact hash manifest"},
	}
}

func BuildReport(reportDir string) (Report, error) {
	reportDir = filepath.Clean(reportDir)
	memory, err := readMemoryReport(filepath.Join(reportDir, "memory-production-linux-x64.json"))
	if err != nil {
		return Report{}, err
	}
	parallel, err := readParallelReport(
		filepath.Join(reportDir, "parallel-production-linux-x64.json"),
	)
	if err != nil {
		return Report{}, err
	}
	ui, err := readUIReport(filepath.Join(reportDir, "ui-production-runtime-linux-x64.json"))
	if err != nil {
		return Report{}, err
	}
	if err := requireManifestSchemas(reportDir, false); err != nil {
		return Report{}, err
	}

	report := Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Target:       "linux-x64",
		CombinedGate: "scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh",
		ReportDir:    reportDir,
		Layers: []LayerReport{
			{
				Name:         "memory",
				Artifact:     "memory-production-linux-x64.json",
				Schema:       memoryprod.SchemaV1,
				Validator:    "go run ./tools/cmd/validate-memory-production --report <path>",
				Status:       memory.Status,
				ProcessCount: len(memory.Processes),
				CaseCount:    len(memory.Cases),
				AuditCount:   len(memory.Audit),
			},
			{
				Name:         "parallelism",
				Artifact:     "parallel-production-linux-x64.json",
				Schema:       parallelprod.SchemaV1,
				Validator:    "go run ./tools/cmd/validate-parallel-production --report <path>",
				Status:       parallel.Status,
				ProcessCount: len(parallel.Processes),
				CaseCount:    len(parallel.Cases),
				AuditCount:   len(parallel.Audit),
			},
			{
				Name:         "ui",
				Artifact:     "ui-production-runtime-linux-x64.json",
				Schema:       uiprod.SchemaV1,
				Validator:    "go run ./tools/cmd/validate-ui-production-runtime --report <path>",
				Status:       ui.Status,
				ProcessCount: len(ui.Processes),
				CaseCount:    len(ui.Cases),
				AuditCount:   len(ui.Audit),
			},
		},
	}
	report.Checklist = append(report.Checklist, memoryChecklist(memory.Audit)...)
	report.Checklist = append(report.Checklist, parallelChecklist(parallel.Audit)...)
	report.Checklist = append(report.Checklist, uiChecklist(ui.Audit)...)
	report.Checklist = append(
		report.Checklist,
		ChecklistItem{
			Layer:       "combined",
			Requirement: "ordered Memory Parallelism UI gate",
			Artifact:    report.CombinedGate,
			Evidence:    "combined gate runs memory, then parallelism, then UI release-gate entrypoints",
			Result:      "pass",
		},
		ChecklistItem{
			Layer:       "combined",
			Requirement: "artifact hash manifest",
			Artifact:    "artifact-hashes.json",
			Evidence: ("manifest lists memory-production-linux-x64.json, parallel-" +
				"production-linux-x64.json, ui-production-runtime-linux-x64.json, and " +
				"native-ui-runtime-linux-x64.integration.json"),
			Result: "pass",
		},
	)
	if err := ValidateReport(report); err != nil {
		return Report{}, err
	}
	if err := validateChecklistArtifactReferences(reportDir, report.Checklist); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ValidateReportDir(reportDir string) error {
	reportDir = filepath.Clean(reportDir)
	raw, err := os.ReadFile(filepath.Join(reportDir, DefaultAuditFilename))
	if err != nil {
		return err
	}
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	if err := ValidateReport(report); err != nil {
		return err
	}
	if _, err := BuildReport(reportDir); err != nil {
		return err
	}
	return requireManifestSchemas(reportDir, true)
}

func ValidateReport(report Report) error {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64", report.Target))
	}
	if strings.TrimSpace(
		report.CombinedGate,
	) != "scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh" {
		issues = append(issues, "combined_gate must name the ordered post-v0.4 production gate")
	}
	if strings.TrimSpace(report.ReportDir) == "" {
		issues = append(issues, "report_dir is required")
	}
	issues = append(issues, validateLayers(report.Layers)...)
	issues = append(issues, validateChecklist(report.Checklist)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateLayers(layers []LayerReport) []string {
	required := map[string]string{
		"memory":      memoryprod.SchemaV1,
		"parallelism": parallelprod.SchemaV1,
		"ui":          uiprod.SchemaV1,
	}
	requiredOrder := []string{"memory", "parallelism", "ui"}
	var issues []string
	seen := map[string]bool{}
	for _, layer := range layers {
		name := strings.TrimSpace(layer.Name)
		if name == "" {
			issues = append(issues, "layer name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate layer %q", name))
		}
		seen[name] = true
		if wantSchema, ok := required[name]; ok && layer.Schema != wantSchema {
			issues = append(
				issues,
				fmt.Sprintf("layer %s schema is %q, want %q", name, layer.Schema, wantSchema),
			)
		}
		if layer.Status != "pass" {
			issues = append(
				issues,
				fmt.Sprintf("layer %s status is %q, want pass", name, layer.Status),
			)
		}
		if strings.TrimSpace(layer.Artifact) == "" {
			issues = append(issues, fmt.Sprintf("layer %s artifact is required", name))
		}
		if strings.TrimSpace(layer.Validator) == "" {
			issues = append(issues, fmt.Sprintf("layer %s validator is required", name))
		}
		if layer.ProcessCount == 0 || layer.CaseCount == 0 || layer.AuditCount == 0 {
			issues = append(
				issues,
				fmt.Sprintf("layer %s must include process, case, and audit counts", name),
			)
		}
	}
	for name := range required {
		if !seen[name] {
			issues = append(issues, fmt.Sprintf("missing layer %q", name))
		}
	}
	for idx, want := range requiredOrder {
		if idx >= len(layers) {
			continue
		}
		if got := strings.TrimSpace(layers[idx].Name); got != want {
			issues = append(
				issues,
				fmt.Sprintf(
					"layers must be ordered memory, parallelism, ui: position %d is %q, want %q",
					idx+1,
					got,
					want,
				),
			)
		}
	}
	return issues
}

func validateChecklist(checklist []ChecklistItem) []string {
	var issues []string
	required := map[string]bool{}
	for _, item := range RequiredChecklist() {
		required[checklistKey(item.Layer, item.Requirement)] = false
	}
	seen := map[string]bool{}
	for _, item := range checklist {
		layer := strings.TrimSpace(item.Layer)
		requirement := strings.TrimSpace(item.Requirement)
		if layer == "" || requirement == "" {
			issues = append(issues, "checklist item layer and requirement are required")
			continue
		}
		key := checklistKey(layer, requirement)
		if seen[key] {
			issues = append(
				issues,
				fmt.Sprintf("duplicate checklist item %s/%s", layer, requirement),
			)
		}
		seen[key] = true
		if _, ok := required[key]; ok {
			required[key] = true
		}
		if strings.TrimSpace(item.Artifact) == "" {
			issues = append(
				issues,
				fmt.Sprintf("checklist item %s/%s artifact is required", layer, requirement),
			)
		}
		if strings.TrimSpace(item.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("checklist item %s/%s evidence is required", layer, requirement),
			)
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(item.Result)), "pass") {
			issues = append(
				issues,
				fmt.Sprintf(
					"checklist item %s/%s result is %q, want pass",
					layer,
					requirement,
					item.Result,
				),
			)
		}
	}
	for _, item := range RequiredChecklist() {
		if !required[checklistKey(item.Layer, item.Requirement)] {
			issues = append(
				issues,
				fmt.Sprintf("missing checklist requirement %s/%s", item.Layer, item.Requirement),
			)
		}
	}
	return issues
}

func checklistKey(layer, requirement string) string {
	return layer + "\x00" + requirement
}

func validateChecklistArtifactReferences(reportDir string, checklist []ChecklistItem) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	var issues []string
	for _, item := range checklist {
		for _, ref := range splitArtifactRefs(item.Artifact) {
			if !isConcreteArtifactRef(ref) {
				continue
			}
			if artifactRefExists(repoRoot, reportDir, ref) {
				continue
			}
			issues = append(
				issues,
				fmt.Sprintf(
					"checklist item %s/%s references missing artifact %s",
					item.Layer,
					item.Requirement,
					ref,
				),
			)
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func splitArtifactRefs(artifact string) []string {
	fields := strings.FieldsFunc(artifact, func(r rune) bool {
		return r == ';' || r == ','
	})
	refs := make([]string, 0, len(fields))
	for _, field := range fields {
		ref := strings.TrimSpace(field)
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func isConcreteArtifactRef(ref string) bool {
	if strings.Contains(ref, "<") || strings.Contains(ref, ">") {
		return false
	}
	if strings.Contains(ref, " ") {
		return false
	}
	if strings.HasPrefix(ref, "-") {
		return false
	}
	return true
}

func artifactRefExists(repoRoot, reportDir, ref string) bool {
	candidates := []string{
		filepath.Join(reportDir, ref),
		filepath.Join(repoRoot, ref),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Clean(candidate)); err == nil {
			return true
		}
	}
	return false
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	var moduleRoot string
	for {
		if _, err := os.Stat(filepath.Join(wd, "AGENTS.md")); err == nil {
			if _, err := os.Stat(filepath.Join(wd, "graphify-out")); err == nil {
				return wd, nil
			}
			if _, err := os.Stat(filepath.Join(wd, "go.work")); err == nil {
				return wd, nil
			}
		}
		if moduleRoot == "" {
			if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
				moduleRoot = wd
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			if moduleRoot != "" {
				return moduleRoot, nil
			}
			return "", errors.New("could not find repo root from current working directory")
		}
		wd = parent
	}
}

func memoryChecklist(audit []memoryprod.AuditReport) []ChecklistItem {
	items := make([]ChecklistItem, 0, len(audit))
	for _, row := range audit {
		items = append(
			items,
			ChecklistItem{
				Layer:       "memory",
				Requirement: row.Requirement,
				Artifact:    row.Artifact,
				Evidence:    row.Evidence,
				Result:      row.Result,
			},
		)
	}
	return items
}

func parallelChecklist(audit []parallelprod.AuditReport) []ChecklistItem {
	items := make([]ChecklistItem, 0, len(audit))
	for _, row := range audit {
		items = append(
			items,
			ChecklistItem{
				Layer:       "parallelism",
				Requirement: row.Requirement,
				Artifact:    row.Artifact,
				Evidence:    row.Evidence,
				Result:      row.Result,
			},
		)
	}
	return items
}

func uiChecklist(audit []uiprod.AuditReport) []ChecklistItem {
	items := make([]ChecklistItem, 0, len(audit))
	for _, row := range audit {
		items = append(
			items,
			ChecklistItem{
				Layer:       "ui",
				Requirement: row.Requirement,
				Artifact:    row.Artifact,
				Evidence:    row.Evidence,
				Result:      row.Result,
			},
		)
	}
	return items
}

func readMemoryReport(path string) (memoryprod.Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return memoryprod.Report{}, err
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		return memoryprod.Report{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	var report memoryprod.Report
	if err := decodeStrict(raw, &report); err != nil {
		return memoryprod.Report{}, err
	}
	return report, nil
}

func readParallelReport(path string) (parallelprod.Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return parallelprod.Report{}, err
	}
	if err := parallelprod.ValidateReport(raw); err != nil {
		return parallelprod.Report{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	var report parallelprod.Report
	if err := decodeStrict(raw, &report); err != nil {
		return parallelprod.Report{}, err
	}
	return report, nil
}

func readUIReport(path string) (uiprod.Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return uiprod.Report{}, err
	}
	if err := uiprod.ValidateReport(raw); err != nil {
		return uiprod.Report{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	var report uiprod.Report
	if err := decodeStrict(raw, &report); err != nil {
		return uiprod.Report{}, err
	}
	return report, nil
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

func requireManifestSchemas(reportDir string, includeAudit bool) error {
	raw, err := os.ReadFile(filepath.Join(reportDir, "artifact-hashes.json"))
	if err != nil {
		return err
	}
	var manifest hashManifest
	if err := decodeStrict(raw, &manifest); err != nil {
		return err
	}
	if manifest.Schema != "tetra.release-artifact-hashes.v1alpha1" {
		return fmt.Errorf(
			"artifact-hashes.json schema is %q, want tetra.release-artifact-hashes.v1alpha1",
			manifest.Schema,
		)
	}
	required := map[string]string{
		"memory-production-linux-x64.json":             memoryprod.SchemaV1,
		"parallel-production-linux-x64.json":           parallelprod.SchemaV1,
		"ui-production-runtime-linux-x64.json":         uiprod.SchemaV1,
		"native-ui-runtime-linux-x64.integration.json": nativeui.SchemaV1,
	}
	if includeAudit {
		required[DefaultAuditFilename] = SchemaV1
	}
	seen := map[string]string{}
	for _, artifact := range manifest.Artifacts {
		seen[artifact.Path] = artifact.Schema
	}
	for path, schema := range required {
		if seen[path] == "" {
			return fmt.Errorf("artifact-hashes.json missing %s", path)
		}
		if seen[path] != schema {
			return fmt.Errorf(
				"artifact-hashes.json schema for %s is %q, want %q",
				path,
				seen[path],
				schema,
			)
		}
	}
	return nil
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
