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
)

type blockContract struct {
	Schema                  string                `json:"schema"`
	Status                  string                `json:"status"`
	SurfaceScope            string                `json:"surface_scope"`
	CorePrimitives          []string              `json:"core_primitives"`
	ForbiddenCorePrimitives []string              `json:"forbidden_core_primitives"`
	ABI                     blockABIContract      `json:"abi"`
	ReportSchemas           map[string]string     `json:"report_schemas"`
	RendererContract        blockRendererContract `json:"renderer_contract"`
	CompatibilityWrappers   []string              `json:"compatibility_wrappers"`
	NonClaims               []string              `json:"nonclaims"`
}

type blockABIContract struct {
	Schema          string   `json:"schema"`
	MaxReturnSlots  int      `json:"max_return_slots"`
	BlockSlots      []string `json:"block_slots"`
	BlockPropsSlots []string `json:"block_props_slots"`
}

type blockRendererContract struct {
	SoftwareRenderer       bool     `json:"software_renderer"`
	AllowedRenderers       []string `json:"allowed_renderers"`
	RequiredReportSections []string `json:"required_report_sections"`
}

type blockContractReport struct {
	Schema                 string            `json:"schema"`
	CorePrimitives         []string          `json:"core_primitives,omitempty"`
	BlockGraph             *json.RawMessage  `json:"block_graph,omitempty"`
	PaintCommands          []json.RawMessage `json:"paint_commands,omitempty"`
	LayoutPasses           []json.RawMessage `json:"layout_passes,omitempty"`
	BlockAccessibilityTree *json.RawMessage  `json:"block_accessibility_tree,omitempty"`
}

var blockForbiddenCorePrimitives = []string{
	"Button",
	"Card",
	"TextField",
	"TextBox",
	"Sidebar",
	"Modal",
}

func main() {
	contractPath := flag.String(
		"contract",
		"",
		"path to tetra.surface.block.contract.v1 contract JSON",
	)
	reportPath := flag.String(
		"report",
		"",
		"optional Surface report JSON to validate against the independent Block contract shape",
	)
	flag.Parse()
	if strings.TrimSpace(*contractPath) == "" && strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --contract or --report is required")
		os.Exit(2)
	}
	if strings.TrimSpace(*contractPath) != "" {
		if err := validateSurfaceBlockContract(*contractPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if strings.TrimSpace(*reportPath) != "" {
		if err := validateSurfaceBlockContractReport(*reportPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func validateSurfaceBlockContract(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var contract blockContract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return err
	}
	return validateSurfaceBlockContractValue(contract)
}

func validateSurfaceBlockContractValue(contract blockContract) error {
	var issues []string
	if contract.Schema != "tetra.surface.block.contract.v1" {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want tetra.surface.block.contract.v1", contract.Schema),
		)
	}
	if contract.Status != "contract-freeze" {
		issues = append(issues, fmt.Sprintf("status is %q, want contract-freeze", contract.Status))
	}
	if contract.SurfaceScope != "surface-block-system-linux-web" {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_scope is %q, want surface-block-system-linux-web",
				contract.SurfaceScope,
			),
		)
	}
	issues = append(
		issues,
		validateBlockCorePrimitiveSet(contract.CorePrimitives, contract.ForbiddenCorePrimitives)...)
	issues = append(issues, validateBlockABIContract(contract.ABI)...)
	issues = append(issues, validateBlockReportSchemas(contract.ReportSchemas)...)
	issues = append(issues, validateBlockRendererContract(contract.RendererContract)...)
	issues = append(issues, validateBlockCompatibilityWrappers(contract.CompatibilityWrappers)...)
	issues = append(issues, validateBlockContractNonClaims(contract.NonClaims)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceBlockContractReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report blockContractReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	var issues []string
	if report.Schema == "" {
		issues = append(issues, "schema is required")
	}
	if len(report.CorePrimitives) > 0 {
		issues = append(
			issues,
			validateBlockCorePrimitiveSet(report.CorePrimitives, blockForbiddenCorePrimitives)...)
	}
	if report.BlockGraph != nil {
		if len(report.PaintCommands) == 0 {
			issues = append(issues, "block_graph report requires paint_commands evidence")
		}
		if len(report.LayoutPasses) == 0 {
			issues = append(issues, "block_graph report requires layout_passes evidence")
		}
		if report.BlockAccessibilityTree == nil {
			issues = append(issues, "block_graph report requires block_accessibility_tree evidence")
		}
	}
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateBlockCorePrimitiveSet(core []string, forbidden []string) []string {
	var issues []string
	if !containsBlockTextFold(core, "Block") {
		issues = append(issues, "core_primitives must include Block")
	}
	for _, primitive := range forbidden {
		if !containsBlockTextFold(forbidden, primitive) {
			issues = append(issues, fmt.Sprintf("forbidden_core_primitives missing %s", primitive))
		}
		if containsBlockTextFold(core, primitive) {
			issues = append(issues, fmt.Sprintf("core_primitives must not include %s", primitive))
		}
	}
	return issues
}

func validateBlockABIContract(abi blockABIContract) []string {
	var issues []string
	if abi.Schema != "tetra.surface.block.abi.v1" {
		issues = append(
			issues,
			fmt.Sprintf("abi.schema is %q, want tetra.surface.block.abi.v1", abi.Schema),
		)
	}
	if abi.MaxReturnSlots != 10 {
		issues = append(
			issues,
			fmt.Sprintf("abi.max_return_slots is %d, want 10", abi.MaxReturnSlots),
		)
	}
	for _, field := range []string{"id", "parent_id", "props"} {
		if !containsBlockTextFold(abi.BlockSlots, field) {
			issues = append(issues, fmt.Sprintf("abi.block_slots missing %s", field))
		}
	}
	if len(abi.BlockPropsSlots) != 8 {
		issues = append(
			issues,
			fmt.Sprintf("abi.block_props_slots length is %d, want 8", len(abi.BlockPropsSlots)),
		)
	}
	for _, field := range []string{
		"layout_mode",
		"paint_layers",
		"text_len",
		"visual_asset",
		"interaction_flags",
		"state_flags",
		"motion_ms",
		"accessibility_role",
	} {
		if !containsBlockTextFold(abi.BlockPropsSlots, field) {
			issues = append(issues, fmt.Sprintf("abi.block_props_slots missing %s", field))
		}
	}
	return issues
}

func validateBlockReportSchemas(schemas map[string]string) []string {
	var issues []string
	required := map[string]string{
		"accessibility_node": "tetra.surface.block-accessibility-node.v1",
		"block_graph":        "tetra.surface.block-graph.v1",
		"layout_pass":        "tetra.surface.layout-pass.v1",
		"paint_command":      "tetra.surface.paint-command.v1",
		"resolved_block":     "tetra.surface.resolved-block.v1",
	}
	for name, want := range required {
		if schemas[name] != want {
			issues = append(
				issues,
				fmt.Sprintf("report_schemas.%s is %q, want %s", name, schemas[name], want),
			)
		}
	}
	return issues
}

func validateBlockRendererContract(contract blockRendererContract) []string {
	var issues []string
	if !contract.SoftwareRenderer {
		issues = append(issues, "renderer_contract software_renderer must be true")
	}
	for _, renderer := range []string{
		"software-rgba-headless",
		"wayland-shm-rgba",
		"browser-canvas-rgba",
	} {
		if !containsBlockTextFold(contract.AllowedRenderers, renderer) {
			issues = append(
				issues,
				fmt.Sprintf("renderer_contract allowed_renderers missing %s", renderer),
			)
		}
	}
	for _, section := range []string{
		"block_graph",
		"paint_commands",
		"layout_passes",
		"block_accessibility_tree",
	} {
		if !containsBlockTextFold(contract.RequiredReportSections, section) {
			issues = append(
				issues,
				fmt.Sprintf("renderer_contract required_report_sections missing %s", section),
			)
		}
	}
	return issues
}

func validateBlockCompatibilityWrappers(wrappers []string) []string {
	var issues []string
	for _, wrapper := range blockForbiddenCorePrimitives {
		if !containsBlockTextFold(wrappers, wrapper) {
			issues = append(issues, fmt.Sprintf("compatibility_wrappers missing %s", wrapper))
		}
	}
	return issues
}

func validateBlockContractNonClaims(nonclaims []string) []string {
	var issues []string
	for _, nonclaim := range []string{
		"no core Button primitive",
		"no CSS layout parity",
		"no GPU renderer production claim",
	} {
		if !containsBlockTextFold(nonclaims, nonclaim) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q", nonclaim))
		}
	}
	return issues
}

func containsBlockTextFold(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}
