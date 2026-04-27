package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type manifestEnvelope struct {
	CompilerVersion string             `json:"compiler_version"`
	TargetsRaw      json.RawMessage    `json:"targets"`
	BuiltinsRaw     json.RawMessage    `json:"builtins"`
	RuntimeABI      runtimeABIManifest `json:"runtime_abi"`
	Targets         []targetManifest
	Builtins        []builtinManifest
}

type targetManifest struct {
	Triple         string `json:"triple"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	ABI            string `json:"abi"`
	Format         string `json:"format"`
	ExeExt         string `json:"exe_ext"`
	CollectImports bool   `json:"collect_imports"`
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
	ReservedPrefix           string   `json:"reserved_prefix"`
	ActorsSupportedTargets   []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols    []string `json:"actors_required_symbols"`
	ActorsProgramGlueSymbols []string `json:"actors_program_glue_symbols"`
}

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
	if !sameStringSet(targetTriples, ctarget.SupportedTriples()) {
		return fmt.Errorf("targets got %s want %s", strings.Join(sortedStrings(targetTriples), ", "), strings.Join(sortedStrings(ctarget.SupportedTriples()), ", "))
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
	return validateRuntimeABI(manifest.RuntimeABI, targets)
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
	return nil
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
	if !sameStringSet(abi.ActorsSupportedTargets, ctarget.SupportedTriples()) {
		return fmt.Errorf("actors_supported_targets got %s want %s", strings.Join(sortedStrings(abi.ActorsSupportedTargets), ", "), strings.Join(sortedStrings(ctarget.SupportedTriples()), ", "))
	}
	requiredRuntimeSymbols := []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_recv",
		"__tetra_actor_self",
		"__tetra_actor_sender",
	}
	if !sameStringSet(abi.ActorsRequiredSymbols, requiredRuntimeSymbols) {
		return fmt.Errorf("actors_required_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsRequiredSymbols), ", "), strings.Join(sortedStrings(requiredRuntimeSymbols), ", "))
	}
	requiredGlueSymbols := []string{
		"__tetra_actor_dispatch",
		"__tetra_actor_main_entry_id",
	}
	if !sameStringSet(abi.ActorsProgramGlueSymbols, requiredGlueSymbols) {
		return fmt.Errorf("actors_program_glue_symbols got %s want %s", strings.Join(sortedStrings(abi.ActorsProgramGlueSymbols), ", "), strings.Join(sortedStrings(requiredGlueSymbols), ", "))
	}
	for _, symbol := range append(append([]string{}, abi.ActorsRequiredSymbols...), abi.ActorsProgramGlueSymbols...) {
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
