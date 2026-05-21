package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type manifestEnvelope struct {
	CompilerVersion string             `json:"compiler_version"`
	FormatsRaw      json.RawMessage    `json:"formats,omitempty"`
	TargetsRaw      json.RawMessage    `json:"targets"`
	BuiltinsRaw     json.RawMessage    `json:"builtins"`
	RuntimeABI      runtimeABIManifest `json:"runtime_abi"`
	FeaturesRaw     json.RawMessage    `json:"features"`
	Formats         []formatManifest
	Targets         []targetManifest
	Builtins        []builtinManifest
	Features        []featureManifest
}

type formatManifest struct {
	Name        string `json:"name"`
	Extension   string `json:"extension,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Primary     bool   `json:"primary,omitempty"`
	Legacy      bool   `json:"legacy,omitempty"`
}

type targetManifest struct {
	Triple                  string `json:"triple"`
	Status                  string `json:"status,omitempty"`
	OS                      string `json:"os"`
	Arch                    string `json:"arch"`
	ABI                     string `json:"abi"`
	DataModel               string `json:"data_model,omitempty"`
	Format                  string `json:"format"`
	ExeExt                  string `json:"exe_ext"`
	CollectImports          bool   `json:"collect_imports"`
	RunMode                 string `json:"run_mode,omitempty"`
	PointerWidthBits        int    `json:"pointer_width_bits,omitempty"`
	RegisterWidthBits       int    `json:"register_width_bits,omitempty"`
	NativeIntWidthBits      int    `json:"native_int_width_bits,omitempty"`
	Endian                  string `json:"endian,omitempty"`
	StackAlignmentBytes     int    `json:"stack_alignment_bytes,omitempty"`
	MaxAtomicWidthBits      int    `json:"max_atomic_width_bits,omitempty"`
	AtomicWidthBits         []int  `json:"atomic_width_bits,omitempty"`
	AtomicPointerWidthBits  int    `json:"atomic_pointer_width_bits,omitempty"`
	UnsupportedReason       string `json:"unsupported_reason,omitempty"`
	SupportsDebugInfo       bool   `json:"supports_debug_info,omitempty"`
	SupportsReleaseOptimize bool   `json:"supports_release_optimize,omitempty"`
}

type builtinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`
	UnsafeDetails string   `json:"unsafe_details,omitempty"`
}

type runtimeABIManifest struct {
	ReservedPrefix            string   `json:"reserved_prefix"`
	ActorsSupportedTargets    []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols     []string `json:"actors_required_symbols"`
	ActorStateRequiredSymbols []string `json:"actor_state_required_symbols,omitempty"`
	TaskRequiredSymbols       []string `json:"task_required_symbols,omitempty"`
	TaskGroupRequiredSymbols  []string `json:"task_group_required_symbols,omitempty"`
	TypedTaskRequiredSymbols  []string `json:"typed_task_required_symbols,omitempty"`
	TimeRequiredSymbols       []string `json:"time_required_symbols,omitempty"`
	FilesystemRequiredSymbols []string `json:"filesystem_required_symbols,omitempty"`
	NetRequiredSymbols        []string `json:"net_required_symbols,omitempty"`
	ActorsProgramGlueSymbols  []string `json:"actors_program_glue_symbols"`
}

type featureManifest struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Since     string   `json:"since,omitempty"`
	Scope     string   `json:"scope"`
	Stability string   `json:"stability"`
	Docs      []string `json:"docs"`
}

const manifestArtifact = "tetra.release.v0_4_0.manifest-json.v1"

func main() {
	var manifestPath string
	flag.StringVar(&manifestPath, "manifest", "", "path to generated manifest JSON")
	flag.Parse()

	if manifestPath == "" {
		fmt.Fprintln(os.Stderr, "error: --manifest is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateManifest(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateManifest(raw []byte) error {
	var manifest manifestEnvelope
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return err
	}
	if manifest.CompilerVersion == "" {
		return fmt.Errorf("compiler_version is required")
	}
	if len(bytes.TrimSpace(manifest.FormatsRaw)) > 0 {
		if err := unmarshalArray(manifest.FormatsRaw, "formats", &manifest.Formats); err != nil {
			return err
		}
		if err := validateFormats(manifest.Formats); err != nil {
			return err
		}
	}
	if err := unmarshalArray(manifest.TargetsRaw, "targets", &manifest.Targets); err != nil {
		return err
	}
	if err := unmarshalArray(manifest.BuiltinsRaw, "builtins", &manifest.Builtins); err != nil {
		return err
	}
	if len(manifest.Targets) == 0 {
		return fmt.Errorf("targets must not be empty")
	}
	if len(manifest.Builtins) == 0 {
		return fmt.Errorf("builtins must not be empty")
	}
	if !isSortedStrings(extractBuiltinNames(manifest.Builtins)) {
		return fmt.Errorf("builtins must be sorted by name for deterministic manifest output")
	}
	targets := map[string]bool{}
	var targetTriples []string
	for _, target := range manifest.Targets {
		if err := validateTarget(target); err != nil {
			return err
		}
		if targets[target.Triple] {
			return fmt.Errorf("duplicate target %s", target.Triple)
		}
		targets[target.Triple] = true
		targetTriples = append(targetTriples, target.Triple)
	}
	supportedTargets := ctarget.SupportedTriples()
	wantTargets := append([]string{}, ctarget.SupportedTriples()...)
	wantTargets = append(wantTargets, ctarget.BuildOnlyTriples()...)
	if !sameStringSet(targetTriples, supportedTargets) && !sameStringSet(targetTriples, wantTargets) {
		return fmt.Errorf("targets got %s want %s", strings.Join(sortedStrings(targetTriples), ", "), strings.Join(sortedStrings(wantTargets), ", "))
	}
	if !sameStringSequence(targetTriples, supportedTargets) && !sameStringSequence(targetTriples, wantTargets) {
		return fmt.Errorf("targets must follow buildable target order: got %s want %s", strings.Join(targetTriples, ", "), strings.Join(wantTargets, ", "))
	}
	builtins := map[string]bool{}
	for _, builtin := range manifest.Builtins {
		if err := validateBuiltin(builtin); err != nil {
			return err
		}
		if builtins[builtin.Name] {
			return fmt.Errorf("duplicate builtin %s", builtin.Name)
		}
		builtins[builtin.Name] = true
	}
	if err := validateRuntimeABI(manifest.RuntimeABI, targets); err != nil {
		return err
	}
	if err := unmarshalArray(manifest.FeaturesRaw, "features", &manifest.Features); err != nil {
		return err
	}
	if featureHasStatus(manifest.Features, "targets.wasm-artifact-preflight", "current") {
		for _, triple := range ctarget.WASMTriples() {
			if !targets[triple] {
				return fmt.Errorf("targets.wasm-artifact-preflight is current but targets missing %s", triple)
			}
		}
	}
	return validateFeatures(manifest.Features)
}

func featureHasStatus(features []featureManifest, id string, status string) bool {
	for _, feature := range features {
		if feature.ID == id && feature.Status == status {
			return true
		}
	}
	return false
}

func validateFormats(formats []formatManifest) error {
	if len(formats) == 0 {
		return fmt.Errorf("formats must not be empty")
	}
	required := map[string]string{
		".t4":        "source",
		".tetra":     "source",
		".tdx":       "todex-fragment",
		".t4s":       "offline-seed",
		".t4i":       "interface",
		".t4p":       "proof",
		".t4r":       "replay",
		".t4q":       "quest",
		".tneed":     "needmap",
		"Tetra.lock": "semantic-lock",
	}
	officialOrder := []string{".t4", ".tetra", ".tdx", ".t4s", ".t4i", ".t4p", ".t4r", ".t4q", ".tneed", "Tetra.lock"}
	seen := map[string]bool{}
	var order []string
	for _, format := range formats {
		if format.Name == "" {
			return fmt.Errorf("format missing name")
		}
		if format.Role == "" {
			return fmt.Errorf("format %s missing role", format.Name)
		}
		if format.Description == "" {
			return fmt.Errorf("format %s missing description", format.Name)
		}
		key := format.Extension
		if key == "" {
			key = format.FileName
		}
		if key == "" {
			return fmt.Errorf("format %s missing extension or file_name", format.Name)
		}
		if format.Extension != "" && format.FileName != "" {
			return fmt.Errorf("format %s must not set both extension and file_name", format.Name)
		}
		if seen[key] {
			return fmt.Errorf("duplicate format %s", key)
		}
		seen[key] = true
		order = append(order, key)
		if wantRole, ok := required[key]; ok && format.Role != wantRole {
			return fmt.Errorf("format %s role = %s, want %s", key, format.Role, wantRole)
		}
		switch key {
		case ".t4":
			if !format.Primary || format.Legacy {
				return fmt.Errorf(".t4 must be primary source format")
			}
		case ".tetra":
			if !format.Legacy || format.Primary {
				return fmt.Errorf(".tetra must be legacy source format")
			}
		}
	}
	for _, key := range officialOrder {
		if !seen[key] {
			return fmt.Errorf("formats missing %s", key)
		}
	}
	if len(order) >= len(officialOrder) && !sameStringSequence(order[:len(officialOrder)], officialOrder) {
		return fmt.Errorf("formats must start with official T4 order: got %s want %s", strings.Join(order[:len(officialOrder)], ", "), strings.Join(officialOrder, ", "))
	}
	return nil
}

func unmarshalArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("%s must be an array", field)
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := decodeStrictJSON(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateTarget(target targetManifest) error {
	if target.Triple == "" {
		return fmt.Errorf("target missing triple")
	}
	if target.OS == "" {
		return fmt.Errorf("target %s missing os", target.Triple)
	}
	if target.Arch == "" {
		return fmt.Errorf("target %s missing arch", target.Triple)
	}
	if target.ABI == "" {
		return fmt.Errorf("target %s missing abi", target.Triple)
	}
	if target.Format == "" {
		return fmt.Errorf("target %s missing format", target.Triple)
	}
	if tgt, err := ctarget.Parse(target.Triple); err == nil {
		if target.Status != "" && target.Status != tgt.Status.String() {
			return fmt.Errorf("target %s status = %s, want %s", target.Triple, target.Status, tgt.Status.String())
		}
		if target.DataModel != "" && target.DataModel != tgt.DataModel.String() {
			return fmt.Errorf("target %s data_model = %s, want %s", target.Triple, target.DataModel, tgt.DataModel.String())
		}
		if target.RunMode != "" && target.RunMode != tgt.RunMode.String() {
			return fmt.Errorf("target %s run_mode = %s, want %s", target.Triple, target.RunMode, tgt.RunMode.String())
		}
		if target.PointerWidthBits != 0 && target.PointerWidthBits != tgt.PointerWidthBits {
			return fmt.Errorf("target %s pointer_width_bits = %d, want %d", target.Triple, target.PointerWidthBits, tgt.PointerWidthBits)
		}
		if target.RegisterWidthBits != 0 && target.RegisterWidthBits != tgt.RegisterWidthBits {
			return fmt.Errorf("target %s register_width_bits = %d, want %d", target.Triple, target.RegisterWidthBits, tgt.RegisterWidthBits)
		}
		if target.NativeIntWidthBits != 0 && target.NativeIntWidthBits != tgt.NativeIntWidthBits {
			return fmt.Errorf("target %s native_int_width_bits = %d, want %d", target.Triple, target.NativeIntWidthBits, tgt.NativeIntWidthBits)
		}
		if target.Endian != "" && target.Endian != tgt.Endian.String() {
			return fmt.Errorf("target %s endian = %s, want %s", target.Triple, target.Endian, tgt.Endian.String())
		}
		if target.StackAlignmentBytes != 0 && target.StackAlignmentBytes != tgt.StackAlignmentBytes {
			return fmt.Errorf("target %s stack_alignment_bytes = %d, want %d", target.Triple, target.StackAlignmentBytes, tgt.StackAlignmentBytes)
		}
		if target.MaxAtomicWidthBits != 0 && target.MaxAtomicWidthBits != tgt.MaxAtomicWidthBits {
			return fmt.Errorf("target %s max_atomic_width_bits = %d, want %d", target.Triple, target.MaxAtomicWidthBits, tgt.MaxAtomicWidthBits)
		}
		if len(target.AtomicWidthBits) > 0 && !sameInts(target.AtomicWidthBits, tgt.AtomicWidthBits()) {
			return fmt.Errorf("target %s atomic_width_bits = %v, want %v", target.Triple, target.AtomicWidthBits, tgt.AtomicWidthBits())
		}
		if target.AtomicPointerWidthBits != 0 {
			ptr, err := tgt.AtomicPointerLayout()
			if err != nil {
				return fmt.Errorf("target %s atomic_pointer_width_bits unsupported: %v", target.Triple, err)
			}
			if target.AtomicPointerWidthBits != ptr.WidthBits {
				return fmt.Errorf("target %s atomic_pointer_width_bits = %d, want %d", target.Triple, target.AtomicPointerWidthBits, ptr.WidthBits)
			}
		}
		if target.UnsupportedReason != "" && target.UnsupportedReason != tgt.UnsupportedReason {
			return fmt.Errorf("target %s unsupported_reason = %q, want %q", target.Triple, target.UnsupportedReason, tgt.UnsupportedReason)
		}
	}
	return nil
}

func sameInts(a []int, b []int) bool {
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

func validateBuiltin(builtin builtinManifest) error {
	if builtin.Name == "" {
		return fmt.Errorf("builtin missing name")
	}
	if builtin.ReturnType == "" {
		return fmt.Errorf("builtin %s missing return_type", builtin.Name)
	}
	switch builtin.UnsafePolicy {
	case "never", "always", "conditional":
	default:
		return fmt.Errorf("builtin %s invalid unsafe_policy %q", builtin.Name, builtin.UnsafePolicy)
	}
	if builtin.UnsafePolicy == "conditional" && builtin.UnsafeDetails == "" {
		return fmt.Errorf("builtin %s conditional unsafe_policy requires unsafe_details", builtin.Name)
	}
	for _, effect := range builtin.Effects {
		if effect == "" {
			return fmt.Errorf("builtin %s has empty effect", builtin.Name)
		}
	}
	return nil
}

func validateFeatures(features []featureManifest) error {
	if len(features) == 0 {
		return fmt.Errorf("features must not be empty")
	}
	allowedStatus := map[string]bool{"current": true, "experimental": true, "planned": true, "post-v1": true}
	requiredStatus := map[string]bool{"current": false, "planned": false, "post-v1": false}
	requiredIDs := map[string]string{
		"cli.core":                                "current",
		"language.flow":                           "current",
		"language.generics-mvp":                   "current",
		"language.protocol-conformance-mvp":       "current",
		"language.callable-mvp":                   "current",
		"language.callable-level1":                "current",
		"targets.wasm-artifact-preflight":         "current",
		"stdlib.experimental-mirrors":             "current",
		"language.enum-payload-match":             "current",
		"language.protocol-bound-generics-static": "current",
		"language.ownership-markers-mvp":          "current",
		"language.resource-lifetime-mvp":          "current",
		"actors.task-transfer-safety":             "current",
		"language.lifetime-ssa":                   "current",
		"safety.production-core":                  "current",
		"language.callable-level2":                "current",
		"ui.metadata-v1":                          "current",
		"wasm.runtime-execution":                  "current",
		"language.full-v1-guarantees":             "planned",
		"eco.distributed-network":                 "post-v1",
		"language.full-first-class-callables":     "current",
	}
	seen := map[string]string{}
	featureByID := map[string]featureManifest{}
	for _, feature := range features {
		if feature.ID == "" {
			return fmt.Errorf("feature missing id")
		}
		if feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			return fmt.Errorf("feature %s missing name, scope, or stability", feature.ID)
		}
		if !allowedStatus[feature.Status] {
			return fmt.Errorf("feature %s invalid status %q", feature.ID, feature.Status)
		}
		if seenStatus, ok := seen[feature.ID]; ok {
			return fmt.Errorf("duplicate feature %s (%s and %s)", feature.ID, seenStatus, feature.Status)
		}
		seen[feature.ID] = feature.Status
		featureByID[feature.ID] = feature
		requiredStatus[feature.Status] = true
		if feature.Status == "current" && feature.Since == "" {
			return fmt.Errorf("current feature %s missing since", feature.ID)
		}
		if len(feature.Docs) == 0 {
			return fmt.Errorf("feature %s missing docs", feature.ID)
		}
		for _, doc := range feature.Docs {
			docPath := filepath.ToSlash(doc)
			if doc == "" || filepath.IsAbs(doc) || strings.Contains(docPath, "..") {
				return fmt.Errorf("feature %s invalid doc reference %q", feature.ID, doc)
			}
			if !strings.HasPrefix(docPath, "docs/") || !strings.HasSuffix(docPath, ".md") {
				return fmt.Errorf("feature %s doc reference %q must point at docs/*.md", feature.ID, doc)
			}
		}
	}
	for status, present := range requiredStatus {
		if !present {
			return fmt.Errorf("features missing %s status", status)
		}
	}
	for id, wantStatus := range requiredIDs {
		if gotStatus, ok := seen[id]; !ok {
			return fmt.Errorf("features missing %s", id)
		} else if gotStatus != wantStatus {
			return fmt.Errorf("feature %s status = %s, want %s", id, gotStatus, wantStatus)
		}
	}
	if err := validateFeatureTruthBoundaries(featureByID); err != nil {
		return err
	}
	return nil
}

func validateFeatureTruthBoundaries(features map[string]featureManifest) error {
	checks := map[string][]string{
		"language.generics-mvp": {
			"statically monomorphized",
			"no runtime generic values or dynamic dispatch",
			"generic structs",
			"future/post-v1",
		},
		"language.protocol-conformance-mvp": {
			"checked statically",
			"generic requirement signature shape",
			"no witness tables",
			"dynamic dispatch remain post-v1",
		},
		"language.ownership-markers-mvp": {
			"conservative borrow/inout/consume marker checks",
			"use-after-consume",
			"borrow escape diagnostics",
			"not a full SSA lifetime solver",
		},
		"language.resource-lifetime-mvp": {
			"conservative resource finalization checks",
			"task handles",
			"island handles",
			"double-use",
			"ambiguous provenance",
			"not a full SSA lifetime solver",
		},
		"actors.task-transfer-safety": {
			"conservative actor/task ownership transfer checks",
			"worker entrypoints",
			"use-after-transfer diagnostics",
			"conservative local MVP",
			"distributed actors",
		},
		"language.lifetime-ssa": {
			"production SSA-like local lifetime join analysis",
			"ownership consume state",
			"resource finalization state",
			"maybe-consumed diagnostics",
			"richer interprocedural lifetime proofs",
		},
		"safety.production-core": {
			"production local safety model",
			"ownership/lifetime/borrow/consume/inout",
			"resource finalization",
			"callable escape diagnostics",
			"effects/capabilities/privacy/consent/budget",
			"unsafe boundaries",
			"actor/task transfer safety",
			"pointer/MMIO/memory capability gates",
			"explicit diagnostics",
		},
		"language.enum-payload-match": {
			"positional enum payload constructors",
			"match/catch/if-let",
			"exhaustive unguarded enum match/catch",
			"nested destructuring patterns",
			"guard expansion remain future/post-v1",
		},
		"language.protocol-bound-generics-static": {
			"validated statically during monomorphization",
			"same-module and cross-module impl conformance",
			"visibility diagnostics",
			"calling protocol requirements through generic bounds",
			"dynamic dispatch remain unsupported",
		},
	}
	docChecks := map[string][]string{
		"language.generics-mvp":                   {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		"language.protocol-conformance-mvp":       {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		"language.ownership-markers-mvp":          {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"language.resource-lifetime-mvp":          {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"actors.task-transfer-safety":             {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"language.lifetime-ssa":                   {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		"safety.production-core":                  {"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md"},
		"language.enum-payload-match":             {"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"},
		"language.protocol-bound-generics-static": {"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"},
	}
	for id, required := range checks {
		feature, ok := features[id]
		if !ok {
			return fmt.Errorf("features missing %s", id)
		}
		haystack := feature.Scope + " " + feature.Stability
		for _, want := range required {
			if !strings.Contains(haystack, want) {
				return fmt.Errorf("feature %s missing truth-boundary phrase %q", id, want)
			}
		}
		for _, doc := range docChecks[id] {
			if !containsString(feature.Docs, doc) {
				return fmt.Errorf("feature %s missing doc reference %s", id, doc)
			}
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateRuntimeABI(abi runtimeABIManifest, targets map[string]bool) error {
	if abi.ReservedPrefix == "" {
		return fmt.Errorf("runtime_abi.reserved_prefix is required")
	}
	if len(abi.ActorsSupportedTargets) == 0 {
		return fmt.Errorf("actors_supported_targets must not be empty")
	}
	if len(abi.ActorsRequiredSymbols) == 0 {
		return fmt.Errorf("actors_required_symbols must not be empty")
	}
	if len(abi.ActorsProgramGlueSymbols) == 0 {
		return fmt.Errorf("actors_program_glue_symbols must not be empty")
	}
	for _, target := range abi.ActorsSupportedTargets {
		if target == "" {
			return fmt.Errorf("actors_supported_targets contains empty target")
		}
		if !targets[target] {
			return fmt.Errorf("actors_supported_targets references unknown target %s", target)
		}
	}
	if !sameStringSet(abi.ActorsSupportedTargets, ctarget.ActorRuntimeTriples()) {
		return fmt.Errorf("actors_supported_targets got %s want %s", strings.Join(sortedStrings(abi.ActorsSupportedTargets), ", "), strings.Join(sortedStrings(ctarget.ActorRuntimeTriples()), ", "))
	}
	requiredRuntimeSymbols := []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	}
	if !sameStringSet(abi.ActorsRequiredSymbols, requiredRuntimeSymbols) {
		return fmt.Errorf("actors_required_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsRequiredSymbols), ", "), strings.Join(sortedStrings(requiredRuntimeSymbols), ", "))
	}
	requiredTimeSymbols := []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
	requiredFilesystemSymbols := []string{
		"__tetra_fs_exists",
	}
	requiredActorStateSymbols := []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	}
	requiredTaskSymbols := []string{
		"__tetra_task_spawn_i32",
		"__tetra_task_join_i32",
		"__tetra_task_join_result_i32",
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	}
	requiredTaskGroupSymbols := []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	}
	requiredTypedTaskSymbols := []string{
		"__tetra_task_result_begin",
		"__tetra_task_result_slot",
		"__tetra_task_result_get",
		"__tetra_task_join_typed_2",
		"__tetra_task_join_typed_3",
		"__tetra_task_join_typed_4",
		"__tetra_task_join_typed_5",
		"__tetra_task_join_typed_6",
		"__tetra_task_join_typed_7",
		"__tetra_task_join_typed_8",
	}
	if len(abi.TimeRequiredSymbols) == 0 {
		return fmt.Errorf("time_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TimeRequiredSymbols, requiredTimeSymbols) {
		return fmt.Errorf("time_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TimeRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTimeSymbols), ", "))
	}
	if len(abi.FilesystemRequiredSymbols) == 0 {
		return fmt.Errorf("filesystem_required_symbols must not be empty")
	}
	if !sameStringSet(abi.FilesystemRequiredSymbols, requiredFilesystemSymbols) {
		return fmt.Errorf("filesystem_required_symbols got %s want %s", strings.Join(sortedStrings(abi.FilesystemRequiredSymbols), ", "), strings.Join(sortedStrings(requiredFilesystemSymbols), ", "))
	}
	if len(abi.ActorStateRequiredSymbols) == 0 {
		return fmt.Errorf("actor_state_required_symbols must not be empty")
	}
	if !sameStringSet(abi.ActorStateRequiredSymbols, requiredActorStateSymbols) {
		return fmt.Errorf("actor_state_required_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorStateRequiredSymbols), ", "), strings.Join(sortedStrings(requiredActorStateSymbols), ", "))
	}
	if len(abi.TaskRequiredSymbols) == 0 {
		return fmt.Errorf("task_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TaskRequiredSymbols, requiredTaskSymbols) {
		return fmt.Errorf("task_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TaskRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTaskSymbols), ", "))
	}
	if len(abi.TaskGroupRequiredSymbols) == 0 {
		return fmt.Errorf("task_group_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TaskGroupRequiredSymbols, requiredTaskGroupSymbols) {
		return fmt.Errorf("task_group_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TaskGroupRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTaskGroupSymbols), ", "))
	}
	if len(abi.TypedTaskRequiredSymbols) == 0 {
		return fmt.Errorf("typed_task_required_symbols must not be empty")
	}
	if !sameStringSet(abi.TypedTaskRequiredSymbols, requiredTypedTaskSymbols) {
		return fmt.Errorf("typed_task_required_symbols got %s want %s", strings.Join(sortedStrings(abi.TypedTaskRequiredSymbols), ", "), strings.Join(sortedStrings(requiredTypedTaskSymbols), ", "))
	}
	requiredGlueSymbols := []string{
		"__tetra_actor_dispatch",
		"__tetra_actor_main_entry_id",
	}
	if !sameStringSet(abi.ActorsProgramGlueSymbols, requiredGlueSymbols) {
		return fmt.Errorf("actors_program_glue_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsProgramGlueSymbols), ", "), strings.Join(sortedStrings(requiredGlueSymbols), ", "))
	}
	allSymbols := append([]string{}, abi.ActorsRequiredSymbols...)
	allSymbols = append(allSymbols, abi.ActorStateRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TaskRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TaskGroupRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TypedTaskRequiredSymbols...)
	allSymbols = append(allSymbols, abi.TimeRequiredSymbols...)
	allSymbols = append(allSymbols, abi.FilesystemRequiredSymbols...)
	allSymbols = append(allSymbols, abi.ActorsProgramGlueSymbols...)
	for _, symbol := range allSymbols {
		if symbol == "" {
			return fmt.Errorf("runtime_abi contains empty symbol")
		}
		if len(symbol) < len(abi.ReservedPrefix) || symbol[:len(abi.ReservedPrefix)] != abi.ReservedPrefix {
			return fmt.Errorf("runtime symbol %s does not use reserved prefix %s", symbol, abi.ReservedPrefix)
		}
	}
	return nil
}

func sameStringSet(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func isSortedStrings(values []string) bool {
	return sort.StringsAreSorted(values)
}

func extractTargetTriples(targets []targetManifest) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Triple)
	}
	return out
}

func extractBuiltinNames(builtins []builtinManifest) []string {
	out := make([]string, 0, len(builtins))
	for _, builtin := range builtins {
		out = append(out, builtin.Name)
	}
	return out
}

func sameStringSequence(a []string, b []string) bool {
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
