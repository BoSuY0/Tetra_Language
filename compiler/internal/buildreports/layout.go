package buildreports

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/layoutopt"
	"tetra_language/compiler/internal/semantics"
)

const P21LayoutPolicy = "p21.0_default_layout_freedom_v1"

var p21LayoutTransforms = []string{
	"field_reordering",
	"padding_removal",
	"hot_cold_splitting",
	"scalar_replacement",
	"aos_to_soa",
}

func BuildLayoutReport(target string, checked *semantics.CheckedProgram) LayoutReport {
	report := LayoutReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 2, Kind: "layout", Target: target},
		Policy:         P21LayoutPolicy,
		Claims: []string{
			"default struct layout is compiler-owned",
			"repr(C) locks layout",
			"public ABI/exported FFI requires explicit repr(C)",
			"layout reports show decisions",
			"Default struct layout is compiler-owned and does not promise C field order, padding, or public ABI layout.",
			"repr(C) locks layout for ABI-facing code and denies field reordering, padding removal, hot/cold splitting, scalar replacement, and AoS-to-SoA transforms.",
			"public ABI/exported FFI requires explicit repr(C).",
			"No field reordering, padding removal, hot/cold splitting, scalar replacement, AoS-to-SoA transform, performance change, or runtime behavior change is claimed by this report.",
		},
	}
	if checked == nil {
		return report
	}
	exported := exportedLayoutABITypeUses(checked)
	for _, st := range checked.Structs {
		if st.Decl == nil {
			continue
		}
		info := checked.Types[st.Name]
		policy := layoutopt.PolicyForStruct(*st.Decl)
		row := LayoutDecisionRow{
			Type:             st.Name,
			Module:           st.Module,
			Repr:             policy.Repr,
			Public:           info != nil && info.Public,
			ABILocked:        policy.ABILocked,
			SourceFieldOrder: sourceFieldOrder(st.Decl),
			PublicABI:        "not_public_abi",
			Reason:           "default struct layout is compiler-owned; public ABI/exported FFI requires explicit repr(C)",
		}
		if info != nil {
			row.CurrentFieldLayout = layoutFieldRows(info.Fields)
		}
		if policy.Repr == frontend.StructReprC {
			row.Decision = "abi_locked_repr_c"
			row.Reason = "repr(C) locks layout; public ABI/exported FFI requires explicit repr(C)"
			row.DeniedTransforms = append([]string(nil), p21LayoutTransforms...)
		} else {
			row.Decision = "compiler_owned_default"
			row.AllowedTransforms = allowedLayoutTransforms(policy)
			row.DeniedTransforms = deniedLayoutTransforms(policy)
		}
		if _, ok := exported[st.Name]; ok {
			report.Summary.ExportedPublicABI++
			if policy.Repr == frontend.StructReprC {
				row.PublicABI = "exported_ffi_explicit_repr_c"
			} else {
				row.PublicABI = "exported_ffi_missing_explicit_repr"
			}
		}
		switch policy.Repr {
		case frontend.StructReprC:
			report.Summary.ReprCABILocked++
		default:
			report.Summary.DefaultCompilerOwned++
		}
		report.Decisions = append(report.Decisions, row)
	}
	sort.Slice(report.Decisions, func(i, j int) bool {
		return report.Decisions[i].Type < report.Decisions[j].Type
	})
	report.Summary.Structs = len(report.Decisions)
	return report
}

func ValidateLayoutReport(report LayoutReport) error {
	if report.SchemaVersion != 2 {
		return fmt.Errorf("layout report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Kind != "layout" {
		return fmt.Errorf("layout report kind = %q, want layout", report.Kind)
	}
	if strings.TrimSpace(report.Target) == "" {
		return fmt.Errorf("layout report target is required")
	}
	if report.Policy != P21LayoutPolicy {
		return fmt.Errorf("layout report policy = %q, want %q", report.Policy, P21LayoutPolicy)
	}
	if report.Summary.Structs != len(report.Decisions) {
		return fmt.Errorf("layout report summary structs = %d, decisions = %d", report.Summary.Structs, len(report.Decisions))
	}
	counts := LayoutSummary{}
	for _, row := range report.Decisions {
		if err := validateLayoutDecisionRow(row); err != nil {
			return err
		}
		counts.Structs++
		switch row.Repr {
		case frontend.StructReprC:
			counts.ReprCABILocked++
		default:
			counts.DefaultCompilerOwned++
		}
		if strings.HasPrefix(row.PublicABI, "exported_ffi") {
			counts.ExportedPublicABI++
		}
	}
	if !reflect.DeepEqual(report.Summary, counts) {
		return fmt.Errorf("layout report summary mismatch: got %+v want %+v", report.Summary, counts)
	}
	for _, claim := range report.Claims {
		if strings.TrimSpace(claim) == "" || containsWeakReportText(claim) {
			return fmt.Errorf("layout report contains weak claim text %q", claim)
		}
	}
	return nil
}

func validateLayoutDecisionRow(row LayoutDecisionRow) error {
	if strings.TrimSpace(row.Type) == "" {
		return fmt.Errorf("layout decision row missing type")
	}
	if strings.TrimSpace(row.Repr) == "" {
		return fmt.Errorf("layout decision row %s missing repr", row.Type)
	}
	if strings.TrimSpace(row.Decision) == "" || strings.TrimSpace(row.PublicABI) == "" || containsWeakReportText(row.Reason) {
		return fmt.Errorf("layout decision row %s has incomplete decision evidence", row.Type)
	}
	switch row.PublicABI {
	case "not_public_abi":
	case "exported_ffi_explicit_repr_c":
		if row.Repr != frontend.StructReprC {
			return fmt.Errorf("layout decision row %s claims exported FFI explicit repr(C) without repr(C)", row.Type)
		}
	case "exported_ffi_missing_explicit_repr":
		return fmt.Errorf("layout decision row %s is exported public ABI without explicit repr(C)", row.Type)
	default:
		return fmt.Errorf("layout decision row %s has unknown public ABI state %q", row.Type, row.PublicABI)
	}
	switch row.Repr {
	case frontend.StructReprC:
		if !row.ABILocked {
			return fmt.Errorf("layout decision row %s repr(C) must be ABI locked", row.Type)
		}
		if row.Decision != "abi_locked_repr_c" {
			return fmt.Errorf("layout decision row %s repr(C) decision = %q", row.Type, row.Decision)
		}
		if len(row.AllowedTransforms) != 0 {
			return fmt.Errorf("layout decision row %s repr(C) must not allow layout transforms", row.Type)
		}
		for _, transform := range p21LayoutTransforms {
			if !stringListContains(row.DeniedTransforms, transform) {
				return fmt.Errorf("layout decision row %s repr(C) missing denied transform %q", row.Type, transform)
			}
		}
	default:
		if row.ABILocked {
			return fmt.Errorf("layout decision row %s default struct must not claim ABI lock", row.Type)
		}
		if row.Decision != "compiler_owned_default" {
			return fmt.Errorf("layout decision row %s default decision = %q", row.Type, row.Decision)
		}
		for _, transform := range p21LayoutTransforms {
			if !stringListContains(row.AllowedTransforms, transform) {
				return fmt.Errorf("layout decision row %s default struct missing allowed transform %q", row.Type, transform)
			}
		}
	}
	return nil
}

func exportedLayoutABITypeUses(checked *semantics.CheckedProgram) map[string]struct{} {
	out := map[string]struct{}{}
	if checked == nil {
		return out
	}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil || fn.Decl.ExportName == "" {
			continue
		}
		sig, ok := checked.FuncSigs[fn.Name]
		if !ok {
			continue
		}
		for _, typ := range sig.ParamTypes {
			collectStructLayoutABITypeUse(typ, checked.Types, out, map[string]bool{})
		}
		collectStructLayoutABITypeUse(sig.ReturnType, checked.Types, out, map[string]bool{})
	}
	return out
}

func collectStructLayoutABITypeUse(typeName string, types map[string]*semantics.TypeInfo, out map[string]struct{}, visiting map[string]bool) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "none" || visiting[typeName] {
		return
	}
	info := types[typeName]
	if info == nil {
		return
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)
	switch info.Kind {
	case semantics.TypeStruct:
		out[typeName] = struct{}{}
		for _, field := range info.Fields {
			collectStructLayoutABITypeUse(field.TypeName, types, out, visiting)
		}
	case semantics.TypeArray, semantics.TypeOptional:
		collectStructLayoutABITypeUse(info.ElemType, types, out, visiting)
	}
}

func sourceFieldOrder(st *frontend.StructDecl) []string {
	if st == nil {
		return nil
	}
	out := make([]string, 0, len(st.Fields))
	for _, field := range st.Fields {
		out = append(out, field.Name)
	}
	return out
}

func layoutFieldRows(fields []semantics.FieldInfo) []LayoutFieldRow {
	out := make([]LayoutFieldRow, 0, len(fields))
	for _, field := range fields {
		out = append(out, LayoutFieldRow{
			Name:      field.Name,
			Type:      field.TypeName,
			Offset:    field.Offset,
			SlotCount: field.SlotCount,
		})
	}
	return out
}

func allowedLayoutTransforms(policy layoutopt.LayoutPolicy) []string {
	out := []string{}
	if policy.MayReorderFields {
		out = append(out, "field_reordering")
	}
	if policy.MayPackFields {
		out = append(out, "padding_removal")
	}
	if policy.MaySplitHotCold {
		out = append(out, "hot_cold_splitting")
	}
	if policy.MayScalarReplace {
		out = append(out, "scalar_replacement")
	}
	if policy.MayTransformAoSToSoA {
		out = append(out, "aos_to_soa")
	}
	return out
}

func deniedLayoutTransforms(policy layoutopt.LayoutPolicy) []string {
	allowed := allowedLayoutTransforms(policy)
	out := []string{}
	for _, transform := range p21LayoutTransforms {
		if !stringListContains(allowed, transform) {
			out = append(out, transform)
		}
	}
	return out
}

func stringListContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsWeakReportText(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return true
	}
	for _, marker := range []string{"todo", "tbd", "placeholder", "fixme"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
