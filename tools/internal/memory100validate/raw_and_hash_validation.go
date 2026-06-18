package memory100validate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func memory100ContainsFold(values []string, want string) bool {
	want = strings.ToLower(want)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), want) {
			return true
		}
	}
	return false
}

func validateMemory100LeakResource(path string, gitHead string) []string {
	var report struct {
		Status  string `json:"status"`
		GitHead string `json:"git_head"`
		Checks  []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			Evidence        string   `json:"evidence"`
			SourceArtifacts []string `json:"source_artifacts"`
		} `json:"checks"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("leak/resource report invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(
			issues,
			fmt.Sprintf("leak/resource report status is %q, want pass", report.Status),
		)
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"leak/resource report git_head %s does not match Memory100 git_head %s",
				report.GitHead,
				gitHead,
			),
		)
	}
	byName := map[string]struct {
		Kind            string
		Evidence        string
		SourceArtifacts []string
	}{}
	for _, check := range report.Checks {
		byName[check.Name] = struct {
			Kind            string
			Evidence        string
			SourceArtifacts []string
		}{
			Kind:            check.Kind,
			Evidence:        check.Evidence,
			SourceArtifacts: check.SourceArtifacts,
		}
	}
	for _, name := range []string{
		"actornet_close_without_cancel",
		"compiler_resource_finalization",
		"surface_frame_escape",
		"actor_task_transfer",
	} {
		check, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("leak/resource report missing check %s", name))
			continue
		}
		if strings.TrimSpace(check.Kind) == "" || strings.TrimSpace(check.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("leak/resource report check %s missing kind or evidence", name),
			)
		}
		if len(nonEmptyMemory100Strings(check.SourceArtifacts)) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("leak/resource report check %s missing source_artifacts", name),
			)
		}
	}
	return issues
}

func validateMemory100SemanticSafetyMatrix(path string, gitHead string) []string {
	var report struct {
		Status  string `json:"status"`
		GitHead string `json:"git_head"`
		Rows    []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			Evidence        string   `json:"evidence"`
			SourceArtifacts []string `json:"source_artifacts"`
			Tests           []string `json:"tests"`
		} `json:"rows"`
		NonClaims []string `json:"non_claims"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("semantic safety matrix invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(
			issues,
			fmt.Sprintf("semantic safety matrix status is %q, want pass", report.Status),
		)
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"semantic safety matrix git_head %s does not match Memory100 git_head %s",
				report.GitHead,
				gitHead,
			),
		)
	}
	byName := map[string]struct {
		Kind            string
		Evidence        string
		SourceArtifacts []string
		Tests           []string
	}{}
	for _, row := range report.Rows {
		byName[row.Name] = struct {
			Kind            string
			Evidence        string
			SourceArtifacts []string
			Tests           []string
		}{
			Kind:            row.Kind,
			Evidence:        row.Evidence,
			SourceArtifacts: row.SourceArtifacts,
			Tests:           row.Tests,
		}
	}
	required := []string{
		"borrowed_view_return_escape",
		"borrowed_view_owned_aggregate_escape",
		"borrowed_text_host_boundary_copy",
		"inout_alias_escape",
		"surface_frame_escape",
		"use_after_present_close",
		"resource_finalizer_double_close",
		"actor_task_non_sendable_transfer",
	}
	for _, name := range required {
		row, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("semantic safety matrix missing row %s", name))
			continue
		}
		if strings.TrimSpace(row.Kind) == "" || strings.TrimSpace(row.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("semantic safety matrix row %s missing kind or evidence", name),
			)
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("semantic safety matrix row %s missing source_artifacts", name),
			)
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("semantic safety matrix row %s missing tests", name),
			)
		}
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "semantic safety matrix missing non_claims")
	}
	return issues
}

func validateMemory100ClaimPolicyArtifact(path string, gitHead string) []string {
	var policy struct {
		Status          string   `json:"status"`
		GitHead         string   `json:"git_head"`
		AllowedClaims   []string `json:"allowed_claims"`
		ForbiddenClaims []string `json:"forbidden_claims"`
		NonClaims       []string `json:"non_claims"`
	}
	if err := readMemory100JSON(path, &policy); err != nil {
		return []string{fmt.Sprintf("claim policy invalid: %v", err)}
	}
	var issues []string
	if policy.Status != "pass" {
		issues = append(issues, fmt.Sprintf("claim policy status is %q, want pass", policy.Status))
	}
	if gitHead != "" && policy.GitHead != gitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"claim policy git_head %s does not match Memory100 git_head %s",
				policy.GitHead,
				gitHead,
			),
		)
	}
	if len(nonEmptyMemory100Strings(policy.AllowedClaims)) == 0 {
		issues = append(issues, "claim policy allowed_claims must not be empty")
	}
	forbidden := nonEmptyMemory100Strings(policy.ForbiddenClaims)
	if len(forbidden) == 0 {
		issues = append(issues, "claim policy forbidden_claims must not be empty")
	}
	for _, want := range []string{
		"Memory is 100% ready",
		"fully proven memory safety",
		"full formal proof of memory safety",
		"all targets memory-stable",
		"all-target memory parity",
		"unsafe/raw memory is safe",
		"no leaks",
	} {
		if !memory100StringSetContains(forbidden, want) {
			issues = append(issues, fmt.Sprintf("claim policy forbidden_claims missing %q", want))
		}
	}
	if len(nonEmptyMemory100Strings(policy.NonClaims)) == 0 {
		issues = append(issues, "claim policy non_claims must not be empty")
	}
	return issues
}

func nonEmptyMemory100Strings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func memory100StringSetContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func memory100GitStatusSnapshotDirty(lines []string) bool {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "## ") {
			continue
		}
		return true
	}
	return false
}

func memory100VerdictClaimsClean(verdict string) bool {
	upper := strings.ToUpper(strings.TrimSpace(verdict))
	for _, marker := range []string{
		"CLEAN",
		"RELEASE_CANDIDATE",
		"PROD_READY_PROVEN",
		"RAW_ACCEPTED_PROVEN_PROD_STABLE_100_PERC",
	} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func validateMemory100HashManifest(hashPath string, reportDir string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("Memory100 hash manifest missing or invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(
			issues,
			fmt.Sprintf(
				"Memory100 hash manifest schema is %q, want %s",
				manifest.Schema,
				memory100HashSchema,
			),
		)
	}
	if manifest.Root != "." {
		issues = append(
			issues,
			fmt.Sprintf("Memory100 hash manifest root is %q, want .", manifest.Root),
		)
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "Memory100 hash manifest artifacts must not be empty")
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(
				issues,
				fmt.Sprintf("Memory100 hash path %q is invalid: %v", artifact.Path, err),
			)
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, "Memory100 hash manifest must not list itself")
			continue
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, "Memory100 hash manifest artifacts must be sorted by path")
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(
				issues,
				fmt.Sprintf("duplicate Memory100 hash entry for %s", artifact.Path),
			)
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(reportDir, artifact.Path)
		if err != nil {
			issues = append(
				issues,
				fmt.Sprintf("hash Memory100 artifact %s: %v", artifact.Path, err),
			)
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(
				issues,
				fmt.Sprintf(
					"size mismatch for %s: got %d want %d",
					artifact.Path,
					actual.Size,
					artifact.Size,
				),
			)
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(
				issues,
				fmt.Sprintf(
					"sha256 mismatch for %s: got %s want %s",
					artifact.Path,
					actual.SHA256,
					artifact.SHA256,
				),
			)
		}
		if actual.Schema != artifact.Schema {
			issues = append(
				issues,
				fmt.Sprintf(
					"schema mismatch for %s: got %q want %q",
					artifact.Path,
					actual.Schema,
					artifact.Schema,
				),
			)
		}
	}
	requiredHashPaths := map[string]bool{"memory-100-prod-stable-manifest.json": true}
	for _, required := range requiredMemory100Artifacts {
		requiredHashPaths[required.Path] = true
	}
	for _, rel := range sortedMemory100Keys(requiredHashPaths) {
		if _, ok := seen[rel]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("missing Memory100 hash manifest entry for %s", rel),
			)
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(reportDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list Memory100 artifacts: %v", err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted Memory100 artifact %s", rel))
			}
		}
	}
	return issues
}

func validateMemory100Claims(label string, values []string, allowNegated bool) []string {
	var issues []string
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, fmt.Sprintf("%s contains empty claim", label))
			continue
		}
		if memory100ContainsForbiddenClaim(value, allowNegated) {
			issues = append(
				issues,
				fmt.Sprintf("%s contains forbidden Memory100 claim: %q", label, value),
			)
		}
	}
	return issues
}

func memory100ContainsForbiddenClaim(value string, allowNegated bool) bool {
	lower := strings.ToLower(value)
	if allowNegated && memory100HasNegation(lower) {
		return false
	}
	for _, phrase := range []string{
		"memory is 100% ready",
		"memory 100% ready",
		"memory is perfect",
		"fully proven memory safety",
		"full formal proof",
		"all targets memory-stable",
		"all targets memory stable",
		"unsafe/raw memory is safe",
		"unsafe memory is safe",
		"raw memory is safe",
		"zero heap for all programs",
		"zero-copy for all programs",
		"zero copy for all programs",
		"production actor runtime",
		"c/rust parity",
		"faster than c",
		"faster than rust",
		"official benchmark result",
		"release accepted",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func memory100HasNegation(lower string) bool {
	for _, marker := range []string{"no ", "not ", "does not ", "without ", "nonclaim", "non-claim"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validateMemory100VerdictDirtyTier(verdict string, dirty bool) []string {
	verdict = strings.TrimSpace(verdict)
	if dirty {
		if verdict != memory100ScopedReadyDirty {
			return []string{
				fmt.Sprintf(
					"dirty Memory100 manifest verdict is %q, want %s",
					verdict,
					memory100ScopedReadyDirty,
				),
			}
		}
		return nil
	}
	if verdict == memory100ScopedReadyDirty {
		return []string{
			fmt.Sprintf(
				"clean Memory100 manifest verdict is %q, want %s or a higher clean evidence tier",
				verdict,
				memory100ScopedReadyLocal,
			),
		}
	}
	return nil
}

func readMemory100StrictJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s must contain a single JSON document", path)
	}
	return nil
}

func readMemory100JSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func validateMemory100SafeRel(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || clean != rel || strings.HasPrefix(clean, "../") ||
		strings.Contains(clean, "/../") {
		return fmt.Errorf("path must be clean and stay under report root")
	}
	return nil
}

func validateMemory100SHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("Memory100 artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("Memory100 artifact %s sha256 must contain 64 hex chars", path)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("Memory100 artifact %s sha256 has non-hex character %q", path, ch)
		}
	}
	return nil
}

func hashMemory100File(root string, rel string) (memory100HashArtifact, error) {
	if err := validateMemory100SafeRel(rel); err != nil {
		return memory100HashArtifact{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return memory100HashArtifact{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return memory100HashArtifact{}, fmt.Errorf("symlink artifact is not allowed")
	}
	if !info.Mode().IsRegular() {
		return memory100HashArtifact{}, fmt.Errorf("artifact is not a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return memory100HashArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return memory100HashArtifact{
		Path:   rel,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectMemory100JSONSchema(raw),
	}, nil
}

func detectMemory100JSONSchema(raw []byte) string {
	var envelope memory100SchemaEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	return memory100SchemaOf(envelope)
}

func memory100SchemaOf(envelope memory100SchemaEnvelope) string {
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}

func isMemory100GitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func sortedMemory100Keys(values map[string]bool) []string {
	var out []string
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func listMemory100ArtifactPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink artifact %s is not allowed", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
