package compiler

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/t4iface"
)

func GenerateInterfaceFile(inputPath string) ([]byte, error) {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	return GenerateInterfaceFromSource(raw, inputPath)
}

func GenerateInterfaceFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if file.Module != "" {
		fmt.Fprintf(&b, "module %s\n\n", file.Module)
	}
	explicitPublic := interfaceUsesExplicitPublic(file)
	imports := interfaceImportsForPublicSurface(file, explicitPublic)
	for _, imp := range imports {
		writeInterfaceImport(&b, imp, explicitPublic)
	}
	if len(imports) > 0 {
		b.WriteByte('\n')
	}
	for _, en := range file.Enums {
		if interfaceDeclPublic(file, en.Public) {
			writeInterfaceEnum(&b, en, explicitPublic)
		}
	}
	for _, st := range file.Structs {
		if interfaceDeclPublic(file, st.Public) {
			writeInterfaceStruct(&b, st.Name, st.Fields, explicitPublic)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		var fields []frontend.FieldDecl
		for _, field := range st.Fields {
			fields = append(fields, frontend.FieldDecl{At: field.At, Name: field.Name, Type: field.Type})
		}
		writeInterfaceStruct(&b, st.Name, fields, explicitPublic)
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "protocol %s:\n", proto.Name)
		for _, req := range proto.Requirements {
			fmt.Fprintf(&b, "    %s\n", formatLSPFuncSigDecl(req))
		}
		b.WriteByte('\n')
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "%s:\n", formatLSPFuncDetail(fn))
		fmt.Fprintf(&b, "    return %s\n\n", interfaceReturnLiteral(fn.ReturnType))
	}
	writeInterfaceHashOnlySurface(&b, file, explicitPublic)
	return t4iface.WithHashHeader(b.Bytes()), nil
}

func InterfaceFingerprintFromSource(src []byte, filename string) (string, error) {
	raw, err := GenerateInterfaceFromSource(src, filename)
	if err != nil {
		return "", err
	}
	hash, _, ok, err := t4iface.SplitHashHeader(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return t4iface.FingerprintBody(raw), nil
	}
	return hash, nil
}

func InterfaceFingerprintFromT4I(raw []byte) (string, error) {
	return t4iface.ValidateHash(raw)
}

func ValidateInterfaceAgainstSource(src []byte, iface []byte, filename string) error {
	expected, err := InterfaceFingerprintFromSource(src, filename)
	if err != nil {
		return err
	}
	actual, err := InterfaceFingerprintFromT4I(iface)
	if err != nil {
		return err
	}
	if expected != actual {
		return fmt.Errorf("%s: public API mismatch: source %s, interface %s", filename, expected, actual)
	}
	return nil
}

func writeInterfaceImport(b *bytes.Buffer, imp frontend.ImportDecl, explicitPublic bool) {
	if explicitPublic && imp.Public {
		b.WriteString("pub ")
	}
	if len(imp.Items) > 0 {
		fmt.Fprintf(b, "import %s.{%s}\n", imp.Path, strings.Join(imp.Items, ", "))
		return
	}
	if imp.Alias != "" {
		fmt.Fprintf(b, "import %s as %s\n", imp.Path, imp.Alias)
	} else {
		fmt.Fprintf(b, "import %s\n", imp.Path)
	}
}

func interfaceUsesExplicitPublic(file *frontend.FileAST) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Public {
			return true
		}
	}
	for _, en := range file.Enums {
		if en.Public {
			return true
		}
	}
	for _, st := range file.Structs {
		if st.Public {
			return true
		}
	}
	for _, st := range file.States {
		if st.Public {
			return true
		}
	}
	for _, view := range file.Views {
		if view.Public {
			return true
		}
	}
	for _, proto := range file.Protocols {
		if proto.Public {
			return true
		}
	}
	for _, ext := range file.Extensions {
		if ext.Public {
			return true
		}
	}
	for _, glob := range file.Globals {
		if glob.Public {
			return true
		}
	}
	for _, fn := range file.Funcs {
		if fn.Public {
			return true
		}
	}
	return false
}

func interfaceDeclPublic(file *frontend.FileAST, public bool) bool {
	if !interfaceUsesExplicitPublic(file) {
		return true
	}
	return public
}

func interfaceImportsForPublicSurface(file *frontend.FileAST, explicitPublic bool) []frontend.ImportDecl {
	if file == nil {
		return nil
	}
	if !explicitPublic {
		return append([]frontend.ImportDecl(nil), file.Imports...)
	}
	refs := interfacePublicTypeRefs(file, explicitPublic)
	out := make([]frontend.ImportDecl, 0, len(file.Imports))
	for _, imp := range file.Imports {
		if imp.Public || interfaceImportUsedByRefs(imp, refs) {
			out = append(out, imp)
		}
	}
	return out
}

func interfacePublicTypeRefs(file *frontend.FileAST, explicitPublic bool) map[string]struct{} {
	refs := map[string]struct{}{}
	add := func(ref frontend.TypeRef) {
		addInterfaceTypeRef(refs, ref)
	}
	for _, en := range file.Enums {
		if !interfaceDeclPublic(file, en.Public) {
			continue
		}
		for _, item := range en.Cases {
			for _, payload := range item.Payload {
				add(payload)
			}
		}
	}
	for _, st := range file.Structs {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		add(view.StateName)
		for _, binding := range view.Bindings {
			add(binding.Type)
		}
		for _, style := range view.Styles {
			add(style.Type)
		}
		for _, item := range view.Accessibility {
			add(item.Type)
		}
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		for _, req := range proto.Requirements {
			addInterfaceFuncSigTypeRefs(refs, req)
		}
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		add(ext.Target)
		for _, method := range ext.Methods {
			addInterfaceFuncTypeRefs(refs, method)
		}
	}
	for _, impl := range file.Impls {
		if !explicitPublic {
			add(impl.Type)
			add(impl.Protocol)
		}
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		add(glob.Type)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		addInterfaceFuncTypeRefs(refs, fn)
	}
	return refs
}

func addInterfaceFuncTypeRefs(refs map[string]struct{}, fn *frontend.FuncDecl) {
	if fn == nil {
		return
	}
	for _, param := range fn.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, fn.ReturnType)
	if fn.HasThrows {
		addInterfaceTypeRef(refs, fn.Throws)
	}
}

func addInterfaceFuncSigTypeRefs(refs map[string]struct{}, sig frontend.FuncSigDecl) {
	for _, param := range sig.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, sig.ReturnType)
	if sig.HasThrows {
		addInterfaceTypeRef(refs, sig.Throws)
	}
}

func addInterfaceTypeRef(refs map[string]struct{}, ref frontend.TypeRef) {
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			addInterfaceTypeRef(refs, *ref.Elem)
		}
	default:
		if ref.Name != "" {
			refs[ref.Name] = struct{}{}
		}
	}
}

func interfaceImportUsedByRefs(imp frontend.ImportDecl, refs map[string]struct{}) bool {
	if len(refs) == 0 {
		return false
	}
	alias := imp.Alias
	if alias == "" {
		alias = lastPathSegment(imp.Path)
	}
	for name := range refs {
		if name == alias || strings.HasPrefix(name, alias+".") || strings.HasPrefix(name, imp.Path+".") {
			return true
		}
		for _, item := range imp.Items {
			if name == item || lastPathSegment(name) == item {
				return true
			}
		}
	}
	return false
}

func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func writeInterfaceHashOnlySurface(b *bytes.Buffer, file *frontend.FileAST, explicitPublic bool) {
	wroteHeader := false
	writeHeader := func() {
		if wroteHeader {
			return
		}
		b.WriteString("// hash-only public surface:\n")
		wroteHeader = true
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// global %s\n", formatLSPGlobalDetail(glob))
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// view %s(%s)\n", view.Name, formatLSPTypeRef(view.StateName))
		for _, binding := range view.Bindings {
			fmt.Fprintf(b, "// view %s binding %s: %s\n", view.Name, binding.Name, formatLSPTypeRef(binding.Type))
		}
		for _, event := range view.Events {
			fmt.Fprintf(b, "// view %s event %s -> %s\n", view.Name, event.Name, event.Command)
		}
		for _, command := range view.Commands {
			fmt.Fprintf(b, "// view %s command %s\n", view.Name, command.Name)
		}
		for _, style := range view.Styles {
			fmt.Fprintf(b, "// view %s style %s: %s\n", view.Name, style.Name, formatLSPTypeRef(style.Type))
		}
		for _, item := range view.Accessibility {
			fmt.Fprintf(b, "// view %s accessibility %s: %s\n", view.Name, item.Name, formatLSPTypeRef(item.Type))
		}
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// extension %s\n", formatLSPTypeRef(ext.Target))
		for _, method := range ext.Methods {
			fmt.Fprintf(b, "// extension method %s\n", formatLSPFuncDetail(method))
		}
	}
	if !explicitPublic {
		for _, impl := range file.Impls {
			writeHeader()
			fmt.Fprintf(b, "// impl %s\n", formatLSPImplName(impl))
		}
	}
	if wroteHeader {
		b.WriteByte('\n')
	}
}

func InterfaceOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	if ext == "" {
		return inputPath + T4InterfaceExtension
	}
	return strings.TrimSuffix(inputPath, ext) + T4InterfaceExtension
}

func writeInterfaceEnum(b *bytes.Buffer, en *frontend.EnumDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "enum %s:\n", en.Name)
	for _, item := range en.Cases {
		if len(item.Payload) == 0 {
			fmt.Fprintf(b, "    case %s\n", item.Name)
			continue
		}
		payloads := make([]string, 0, len(item.Payload))
		for _, payload := range item.Payload {
			payloads = append(payloads, formatLSPTypeRef(payload))
		}
		fmt.Fprintf(b, "    case %s(%s)\n", item.Name, strings.Join(payloads, ", "))
	}
	b.WriteByte('\n')
}

func writeInterfaceStruct(b *bytes.Buffer, name string, fields []frontend.FieldDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "struct %s:\n", name)
	if len(fields) == 0 {
		b.WriteString("    _empty: Int\n\n")
		return
	}
	for _, field := range fields {
		fmt.Fprintf(b, "    %s: %s\n", field.Name, formatLSPTypeRef(field.Type))
	}
	b.WriteByte('\n')
}

func interfaceReturnLiteral(ref frontend.TypeRef) string {
	name := canonicalLSPTypeName(formatLSPTypeRef(ref))
	switch name {
	case "bool":
		return "false"
	case "str":
		return "\"\""
	default:
		return "0"
	}
}
