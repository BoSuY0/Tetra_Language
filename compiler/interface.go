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
			writeInterfaceStruct(&b, st.Name, st.TypeParams, st.Fields, explicitPublic)
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
		writeInterfaceStruct(&b, st.Name, nil, fields, explicitPublic)
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
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		writeInterfaceExtension(&b, ext, explicitPublic)
	}
	for _, impl := range file.Impls {
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		writeInterfaceImpl(&b, impl)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "%s:\n", formatLSPFuncDetail(fn))
		fmt.Fprintf(&b, "%s\n\n", interfaceFunctionBody(fn))
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
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		add(impl.Type)
		add(impl.Protocol)
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
	for _, bound := range fn.TypeParamBounds {
		addInterfaceTypeRef(refs, bound.Bound)
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
	for _, arg := range ref.TypeArgs {
		addInterfaceTypeRef(refs, arg)
	}
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			addInterfaceTypeRef(refs, *ref.Elem)
		}
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			addInterfaceTypeRef(refs, param)
		}
		if ref.Return != nil {
			addInterfaceTypeRef(refs, *ref.Return)
		}
		if ref.Throws != nil {
			addInterfaceTypeRef(refs, *ref.Throws)
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
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		source, ok := interfaceBorrowedReturnExpr(fn)
		if !ok {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// func %s lifetime return=borrow source=%s provenance=param lifetime=call\n", formatLSPFuncDetail(fn), source)
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

func writeInterfaceStruct(b *bytes.Buffer, name string, typeParams []string, fields []frontend.FieldDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "struct %s%s:\n", name, formatLSPTypeParams(typeParams, nil))
	if len(fields) == 0 {
		b.WriteString("    _empty: Int\n\n")
		return
	}
	for _, field := range fields {
		fmt.Fprintf(b, "    %s: %s\n", field.Name, formatLSPTypeRef(field.Type))
	}
	b.WriteByte('\n')
}

func writeInterfaceExtension(b *bytes.Buffer, ext *frontend.ExtensionDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	target := formatLSPTypeRef(ext.Target)
	fmt.Fprintf(b, "extension %s:\n", target)
	for _, method := range ext.Methods {
		if method == nil || method.Synthetic {
			continue
		}
		fmt.Fprintf(b, "    %s:\n", formatInterfaceExtensionMethodDetail(method, target))
		body := strings.TrimSuffix(interfaceFunctionBody(method), "\n")
		for _, line := range strings.Split(body, "\n") {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
	b.WriteByte('\n')
}

func formatInterfaceExtensionMethodDetail(fn *frontend.FuncDecl, target string) string {
	if fn == nil {
		return ""
	}
	copyFn := *fn
	prefix := target + "."
	if strings.HasPrefix(copyFn.Name, prefix) {
		copyFn.Name = strings.TrimPrefix(copyFn.Name, prefix)
	}
	return formatLSPFuncDetail(&copyFn)
}

func writeInterfaceImpl(b *bytes.Buffer, impl *frontend.ImplDecl) {
	fmt.Fprintf(b, "impl %s\n\n", formatLSPImplName(impl))
}

func interfaceImplPublic(file *frontend.FileAST, impl *frontend.ImplDecl, explicitPublic bool) bool {
	if impl == nil {
		return false
	}
	if !explicitPublic {
		return true
	}
	return interfaceTypeRefPublic(file, impl.Type) && interfaceTypeRefPublic(file, impl.Protocol)
}

func interfaceTypeRefPublic(file *frontend.FileAST, ref frontend.TypeRef) bool {
	if file == nil || ref.Name == "" {
		return true
	}
	name := ref.Name
	if file.Module != "" {
		name = strings.TrimPrefix(name, file.Module+".")
	}
	for _, en := range file.Enums {
		if en.Name == name {
			return interfaceDeclPublic(file, en.Public)
		}
	}
	for _, st := range file.Structs {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, st := range file.States {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, proto := range file.Protocols {
		if proto.Name == name {
			return interfaceDeclPublic(file, proto.Public)
		}
	}
	return true
}

func interfaceReturnLiteral(ref frontend.TypeRef) string {
	if ref.Kind == frontend.TypeRefOptional {
		return "none"
	}
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

func interfaceReturnExpr(fn *frontend.FuncDecl) string {
	if expr, ok := interfaceTryReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalParamReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalAggregateReturnExpr(fn); ok {
		return expr
	}
	if fn.ReturnType.Kind == frontend.TypeRefFunction {
		if expr, ok := interfaceFunctionReturnExpr(fn); ok {
			return expr
		}
		return interfaceFunctionClosureLiteral(fn.ReturnType, "        ")
	}
	if expr, ok := interfaceBorrowedReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceAggregateReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceSameTypedParameterReturnExpr(fn); ok {
		return expr
	}
	return interfaceReturnLiteral(fn.ReturnType)
}

func interfaceTryReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if _, ok := ret.Value.(*frontend.TryExpr); !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(ret.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(ret.Value)
		}
		if !ok || (!interfaceExprRefsAnyParam(ret.Value, paramNames) && !interfaceExprRefsAnyAlias(ret.Value, aliases)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceOptionalParamReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional || fn.ReturnType.Elem == nil {
		return "", false
	}
	elemType := formatLSPTypeRef(*fn.ReturnType.Elem)
	paramNames := map[string]bool{}
	paramHasElemType := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
		if formatLSPTypeRef(param.Type) == elemType {
			paramHasElemType[param.Name] = true
		}
	}
	optionalLocals := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == formatLSPTypeRef(fn.ReturnType) {
				optionalLocals[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramHasElemType[value.Name] {
					optionalLocals[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames); ok {
					optionalLocals[s.Name] = value
				}
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalLocals[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			optionalLocals[target.Name] = value
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if paramName := optionalLocals[id.Name]; paramName != "" {
				return paramName, true
			}
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(optionalLocals)
			branchAliases[s.Name] = value
			if expr, ok := interfaceOptionalReturnFromStmts(s.Then, branchAliases); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			for _, c := range s.Cases {
				name, ok := interfaceOptionalSomePatternName(c.Pattern)
				if !ok {
					continue
				}
				branchAliases := interfaceAliasMapCopy(optionalLocals)
				branchAliases[name] = value
				if expr, ok := interfaceOptionalReturnFromStmts(c.Body, branchAliases); ok {
					return expr, true
				}
			}
		}
	}
	return "", false
}

func interfaceOptionalAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	optionalAggregates := map[string]string{}
	return interfaceOptionalAggregateReturnFromStmts(fn.Body, aliases, optionalAggregates, paramNames, formatLSPTypeRef(fn.ReturnType))
}

func interfaceOptionalAggregateReturnFromStmts(stmts []frontend.Stmt, aliases, optionalAggregates map[string]string, params map[string]bool, returnType string) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == returnType {
				optionalAggregates[s.Name] = ""
				if value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params); ok {
					optionalAggregates[s.Name] = value
				}
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[s.Name] = value
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalAggregates[target.Name]; ok {
				value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params)
				if ok {
					optionalAggregates[target.Name] = value
				}
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[target.Name] = value
			}
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if value := optionalAggregates[id.Name]; value != "" {
				return value, true
			}
		case *frontend.IfStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(s.Then, thenAliases, thenAggregates, params, returnType)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(s.Else, elseAliases, elseAggregates, params, returnType)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.IfLetStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok && s.Name != "" {
				thenAliases[s.Name] = value
			}
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(s.Then, thenAliases, thenAggregates, params, returnType)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(s.Else, elseAliases, elseAggregates, params, returnType)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceOptionalAggregateMatchReturnExpr(s, aliases, optionalAggregates, params, returnType); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceBorrowedReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnOwnership != "borrow" {
		return "", false
	}
	params := map[string]bool{}
	paramOrder := []string{}
	returnType := formatLSPTypeRef(fn.ReturnType)
	for _, param := range fn.Params {
		if param.Ownership == "borrow" {
			params[param.Name] = true
			if formatLSPTypeRef(param.Type) == returnType {
				paramOrder = append(paramOrder, param.Name)
			}
		}
	}
	if len(params) == 0 {
		return "", false
	}
	for _, stmt := range fn.Body {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if source, ok := interfaceBorrowedSourceParamExpr(ret.Value, params); ok {
			return source, true
		}
	}
	if len(paramOrder) == 1 {
		return paramOrder[0], true
	}
	return "", false
}

func interfaceBorrowedSourceParamExpr(expr frontend.Expr, params map[string]bool) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		return interfaceBorrowedSourceParamExpr(e.Base, params)
	case *frontend.CallExpr:
		method, ok := interfaceBorrowedViewMethod(e.Name)
		if ok && len(e.Args) > 0 {
			switch method {
			case "borrow", "window", "prefix", "suffix":
				return interfaceBorrowedSourceParamExpr(e.Args[0], params)
			}
		}
		return interfaceBorrowedSourceParamMethodCall(e.Name, params)
	}
	return "", false
}

func interfaceBorrowedSourceParamMethodCall(name string, params map[string]bool) (string, bool) {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return "", false
	}
	receiver := name[:idx]
	method := name[idx+1:]
	switch method {
	case "borrow", "window", "prefix", "suffix":
	default:
		return "", false
	}
	root := receiver
	if dot := strings.Index(root, "."); dot >= 0 {
		root = root[:dot]
	}
	if params[root] {
		return root, true
	}
	return "", false
}

func interfaceBorrowedViewMethod(name string) (string, bool) {
	if strings.HasPrefix(name, "__method.") {
		method := strings.TrimPrefix(name, "__method.")
		switch method {
		case "borrow", "window", "prefix", "suffix":
			return method, true
		}
	}
	if name == "core.string_borrow" {
		return "borrow", true
	}
	if strings.HasPrefix(name, "core.slice_borrow_") {
		return "borrow", true
	}
	if name == "core.string_window" || name == "core.string_prefix" || name == "core.string_suffix" {
		return strings.TrimPrefix(name, "core.string_"), true
	}
	if strings.HasPrefix(name, "core.slice_window_") {
		return "window", true
	}
	if strings.HasPrefix(name, "core.slice_prefix_") {
		return "prefix", true
	}
	if strings.HasPrefix(name, "core.slice_suffix_") {
		return "suffix", true
	}
	return "", false
}

func interfaceOptionalAggregateMatchReturnExpr(match *frontend.MatchStmt, aliases, optionalAggregates map[string]string, params map[string]bool, returnType string) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	allReturned := true
	caseAliases := make([]map[string]string, 0, len(match.Cases))
	caseAggregates := make([]map[string]string, 0, len(match.Cases))
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		branchAliases := interfaceAliasMapCopy(aliases)
		branchAggregates := interfaceAliasMapCopy(optionalAggregates)
		expr, returned := interfaceOptionalAggregateReturnFromStmts(c.Body, branchAliases, branchAggregates, params, returnType)
		if returned {
			if commonExpr == "" {
				commonExpr = expr
			} else if commonExpr != expr {
				return "", false
			}
		} else {
			allReturned = false
		}
		caseAliases = append(caseAliases, branchAliases)
		caseAggregates = append(caseAggregates, branchAggregates)
	}
	if allReturned && commonExpr != "" {
		return commonExpr, true
	}
	if !allReturned && commonExpr == "" {
		interfaceMergeEqualAliasStateAcross(aliases, caseAliases)
		interfaceMergeEqualOptionalAggregateStateAcross(optionalAggregates, caseAggregates)
	}
	return "", false
}

func interfaceMergeEqualAliasState(dst, left, right map[string]string) {
	for key := range dst {
		if left[key] == right[key] {
			dst[key] = left[key]
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateState(dst, left, right map[string]string) {
	for key := range dst {
		value, ok := interfaceMergeOptionalAggregateValue(left[key], right[key])
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceMergeOptionalAggregateValue(values ...string) (string, bool) {
	merged := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if merged == "" {
			merged = value
			continue
		}
		if merged != value {
			return "", false
		}
	}
	return merged, true
}

func interfaceMergeEqualAliasStateAcross(dst map[string]string, states []map[string]string) {
	for key := range dst {
		value, ok := interfaceCommonStateValue(key, states)
		if ok {
			dst[key] = value
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateStateAcross(dst map[string]string, states []map[string]string) {
	for key := range dst {
		values := make([]string, 0, len(states))
		for _, state := range states {
			values = append(values, state[key])
		}
		value, ok := interfaceMergeOptionalAggregateValue(values...)
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceCommonStateValue(key string, states []map[string]string) (string, bool) {
	if len(states) == 0 {
		return "", false
	}
	value, ok := states[0][key]
	if !ok {
		return "", false
	}
	for _, state := range states[1:] {
		if state[key] != value {
			return "", false
		}
	}
	return value, true
}

func interfaceOptionalAggregateExpr(expr frontend.Expr, aliases map[string]string, params map[string]bool) (string, bool) {
	if !interfaceDirectAggregateExpr(expr) {
		return "", false
	}
	formatted, ok := interfaceAggregateStubExprWithAliases(expr, aliases)
	if !ok {
		return "", false
	}
	if !interfaceExprRefsAnyParam(expr, params) && !interfaceExprRefsAnyAlias(expr, aliases) {
		return "", false
	}
	return formatted, true
}

func interfaceOptionalReturnFromStmts(stmts []frontend.Stmt, aliases map[string]string) (string, bool) {
	for _, stmt := range stmts {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if !ok {
			continue
		}
		if paramName := aliases[id.Name]; paramName != "" {
			return paramName, true
		}
	}
	return "", false
}

func interfaceFunctionBody(fn *frontend.FuncDecl) string {
	if body, ok := interfaceFunctionMatchReturnBody(fn); ok {
		return body
	}
	if body, ok := interfaceReturnedClosureCaptureBody(fn); ok {
		return body
	}
	if expr, ok := interfaceThrowExpr(fn); ok {
		return "    throw " + expr
	}
	if body, ok := interfaceBorrowedReturnBody(fn); ok {
		return body
	}
	return "    return " + interfaceReturnExpr(fn)
}

func interfaceBorrowedReturnBody(fn *frontend.FuncDecl) (string, bool) {
	source, ok := interfaceBorrowedReturnExpr(fn)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("    // tetra-interface-lifetime: return=borrow source=%s provenance=param lifetime=call\n    return %s", source, source), true
}
