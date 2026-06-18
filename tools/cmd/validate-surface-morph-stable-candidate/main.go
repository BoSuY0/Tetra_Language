package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"tetra_language/tools/validators/surface"
)

type morphStableCandidateContract struct {
	Schema                 string                       `json:"schema"`
	Status                 string                       `json:"status"`
	CurrentTier            string                       `json:"current_tier"`
	TargetTier             string                       `json:"target_tier"`
	SurfaceScope           string                       `json:"surface_scope"`
	ProductionClaim        bool                         `json:"production_claim"`
	ValidatorEnabled       bool                         `json:"validator_enabled"`
	DisabledUntil          string                       `json:"disabled_until"`
	RequiredTargetEvidence []string                     `json:"required_target_evidence"`
	StableSchemas          map[string]morphStableSchema `json:"stable_schemas"`
	RecipeContract         morphStableRecipeContract    `json:"recipe_contract"`
	PromotionGates         []string                     `json:"promotion_gates"`
	NonClaims              []string                     `json:"nonclaims"`
}

type morphStableSchema struct {
	Schema                string   `json:"schema"`
	RequiredFields        []string `json:"required_fields"`
	BackwardCompatibility []string `json:"backward_compatibility"`
}

type morphStableRecipeContract struct {
	AllowedOutputs                   []string `json:"allowed_outputs"`
	ForbiddenOutputs                 []string `json:"forbidden_outputs"`
	RequiresExpandsToBlockGraph      bool     `json:"requires_expands_to_block_graph"`
	RequiresNoHiddenAppState         bool     `json:"requires_no_hidden_app_state"`
	RequiresNoPlatformWidgets        bool     `json:"requires_no_platform_widgets"`
	RequiresNoCorePrimitivePromotion bool     `json:"requires_no_core_primitive_promotion"`
}

var requiredMorphStableSchemas = map[string][]string{
	"accessibility_projection": {
		"schema",
		"derived_from_block_graph",
		"safety_overrides_win",
		"snapshot_evidence",
		"required_fields",
		"roles",
	},
	"affordance": {
		"name",
		"role",
		"focusable",
		"action",
		"input",
		"projects_accessibility",
	},
	"material": {
		"name",
		"paint_stack",
		"fill",
		"border",
		"radius",
		"shadow",
		"overlay",
	},
	"motion_preset": {
		"name",
		"duration_ms",
		"curve",
		"properties",
		"reduced_motion",
		"deterministic_time",
	},
	"recipe":     {"name", "output", "slots", "inputs", "expands_to_block_graph"},
	"state_lens": {"selector", "property", "deterministic"},
	"token_graph": {
		"schema",
		"namespace",
		"version",
		"hash",
		"source_of_truth",
		"explicit_imports",
		"no_global_cascade",
		"fixed_override_order",
		"categories",
		"tokens",
		"density_dpi",
		"diagnostics",
	},
	"variant": {"name", "state_lenses", "materials", "motion"},
}

func main() {
	contractPath := flag.String(
		"contract",
		"",
		"path to tetra.surface.morph.stable-candidate.v1 contract JSON",
	)
	flag.Parse()
	if strings.TrimSpace(*contractPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --contract is required")
		os.Exit(2)
	}
	if err := validateSurfaceMorphStableCandidate(*contractPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceMorphStableCandidate(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var contract morphStableCandidateContract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return err
	}
	return validateSurfaceMorphStableCandidateContract(contract)
}

func validateSurfaceMorphStableCandidateContract(contract morphStableCandidateContract) error {
	var issues []string
	if contract.Schema != "tetra.surface.morph.stable-candidate.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"schema is %q, want tetra.surface.morph.stable-candidate.v1",
				contract.Schema,
			),
		)
	}
	if contract.Status != "design-freeze" {
		issues = append(issues, fmt.Sprintf("status is %q, want design-freeze", contract.Status))
	}
	if !surface.ValidSurfaceClaimTier(contract.CurrentTier) ||
		contract.CurrentTier != string(surface.ClaimTierExperimental) {
		issues = append(
			issues,
			fmt.Sprintf(
				"current_tier is %q, want %s",
				contract.CurrentTier,
				surface.ClaimTierExperimental,
			),
		)
	}
	if !surface.ValidSurfaceClaimTier(contract.TargetTier) ||
		contract.TargetTier != string(surface.ClaimTierProdStableScoped) {
		issues = append(
			issues,
			fmt.Sprintf(
				"target_tier is %q, want %s",
				contract.TargetTier,
				surface.ClaimTierProdStableScoped,
			),
		)
	}
	if contract.SurfaceScope != surface.ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_scope is %q, want %s",
				contract.SurfaceScope,
				surface.ReleaseScopeSurfaceV1LinuxWeb,
			),
		)
	}
	if contract.ValidatorEnabled {
		issues = append(
			issues,
			"stable Morph promotion validator must remain disabled until P20+ evidence exists",
		)
	}
	if !strings.Contains(strings.ToUpper(contract.DisabledUntil), "P20") {
		issues = append(issues, "disabled_until must name P20+")
	}
	issues = append(issues, validateMorphStableTargetEvidence(contract)...)
	issues = append(issues, validateMorphStableSchemas(contract.StableSchemas)...)
	issues = append(issues, validateMorphStableRecipeContract(contract.RecipeContract)...)
	issues = append(issues, validateMorphStablePromotionGates(contract.PromotionGates)...)
	issues = append(issues, validateMorphStableNonClaims(contract.NonClaims)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMorphStableTargetEvidence(contract morphStableCandidateContract) []string {
	var issues []string
	required := []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}
	for _, target := range required {
		if !containsTextFold(contract.RequiredTargetEvidence, target) {
			issues = append(issues, fmt.Sprintf("required target evidence missing %s", target))
		}
	}
	if contract.ProductionClaim && len(issues) > 0 {
		issues = append(issues, "production Morph claim requires complete target evidence")
	}
	return issues
}

func validateMorphStableSchemas(schemas map[string]morphStableSchema) []string {
	var issues []string
	if len(schemas) == 0 {
		return []string{"stable_schemas are required"}
	}
	for name, fields := range requiredMorphStableSchemas {
		schema, ok := schemas[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("stable_schemas missing %s", name))
			continue
		}
		if strings.TrimSpace(schema.Schema) == "" ||
			!strings.HasPrefix(schema.Schema, "tetra.surface.morph.") {
			issues = append(issues, fmt.Sprintf("stable_schemas.%s schema is invalid", name))
		}
		for _, field := range fields {
			if !containsTextFold(schema.RequiredFields, field) {
				issues = append(
					issues,
					fmt.Sprintf("stable_schemas.%s required_fields missing %s", name, field),
				)
			}
		}
		for _, compat := range []string{"versioned_schema", "additive_fields_only"} {
			if !containsTextFold(schema.BackwardCompatibility, compat) {
				issues = append(
					issues,
					fmt.Sprintf(
						"stable_schemas.%s backward_compatibility missing %s",
						name,
						compat,
					),
				)
			}
		}
	}
	return issues
}

func validateMorphStableRecipeContract(contract morphStableRecipeContract) []string {
	var issues []string
	if !sameStringSetFold(contract.AllowedOutputs, []string{"Block"}) {
		issues = append(issues, "recipe_contract allowed_outputs must be exactly [Block]")
	}
	for _, forbidden := range []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"} {
		if !containsTextFold(contract.ForbiddenOutputs, forbidden) {
			issues = append(
				issues,
				fmt.Sprintf("recipe_contract forbidden_outputs missing %s", forbidden),
			)
		}
		if containsTextFold(contract.AllowedOutputs, forbidden) {
			issues = append(
				issues,
				fmt.Sprintf("recipe_contract allowed_outputs must not include %s", forbidden),
			)
		}
	}
	if !contract.RequiresExpandsToBlockGraph {
		issues = append(issues, "recipe_contract requires_expands_to_block_graph must be true")
	}
	if !contract.RequiresNoHiddenAppState {
		issues = append(issues, "recipe_contract requires_no_hidden_app_state must be true")
	}
	if !contract.RequiresNoPlatformWidgets {
		issues = append(issues, "recipe_contract requires_no_platform_widgets must be true")
	}
	if !contract.RequiresNoCorePrimitivePromotion {
		issues = append(issues, "recipe_contract requires_no_core_primitive_promotion must be true")
	}
	return issues
}

func validateMorphStablePromotionGates(gates []string) []string {
	var issues []string
	for _, gate := range []string{
		"validate-surface-morph-report",
		"validate-surface-claims",
		"visual regression gate",
		"target-host evidence",
		"renderer-owned stable proof",
	} {
		if !containsTextFold(gates, gate) {
			issues = append(issues, fmt.Sprintf("promotion_gates missing %q", gate))
		}
	}
	return issues
}

func validateMorphStableNonClaims(nonclaims []string) []string {
	var issues []string
	for _, nonclaim := range []string{
		"not production Morph today",
		"no React runtime",
		"no Electron runtime",
		"no CSS cascade runtime",
		"no platform-native widgets",
	} {
		if !containsTextFold(nonclaims, nonclaim) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q", nonclaim))
		}
	}
	return issues
}

func containsTextFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func sameStringSetFold(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, value := range want {
		if !containsTextFold(got, value) {
			return false
		}
	}
	return true
}
