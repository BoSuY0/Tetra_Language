package semantics

import (
	"encoding/binary"
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

type CheckedProgram struct {
	Funcs              []CheckedFunc
	Structs            []CheckedStruct
	FuncSigs           map[string]FuncSig
	Types              map[string]*TypeInfo
	GlobalsByModule    map[string]map[string]GlobalInfo
	GlobalDataByModule map[string][][]byte
	MainIndex          int
	MainName           string
}

type CheckedFunc struct {
	Name        string
	Module      string
	Decl        *frontend.FuncDecl
	Locals      map[string]LocalInfo
	LocalSlots  int
	ParamSlots  int
	ReturnType  string
	ReturnSlots int
}

type LocalInfo struct {
	Base      int
	SlotCount int
	TypeName  string
	Mutable   bool
}

type GlobalInfo struct {
	DataIndex int
	TypeName  string
	Mutable   bool
}

type FuncSig struct {
	ParamTypes        []string
	ParamSlots        int
	ReturnType        string
	ReturnSlots       int
	ReturnRegionParam int
}

type CheckedStruct struct {
	Name   string
	Module string
	Decl   *frontend.StructDecl
}

type TypeKind int

const (
	TypeI32 TypeKind = iota
	TypeU8
	TypePtr
	TypeSlice
	TypeStr
	TypeStruct
	TypeArray
	TypeIsland
	TypeCap
	TypeActor
)

type FieldInfo struct {
	Name      string
	TypeName  string
	Offset    int
	SlotCount int
}

type TypeInfo struct {
	Name      string
	Kind      TypeKind
	Fields    []FieldInfo
	FieldMap  map[string]FieldInfo
	SlotCount int
	ElemType  string
	ArrayLen  int
}

func makeSliceTypeInfo(name, elem string) *TypeInfo {
	fieldMap := map[string]FieldInfo{
		"ptr": {Name: "ptr", TypeName: "ptr", Offset: 0, SlotCount: 1},
		"len": {Name: "len", TypeName: "i32", Offset: 1, SlotCount: 1},
	}
	fields := []FieldInfo{fieldMap["ptr"], fieldMap["len"]}
	return &TypeInfo{
		Name:      name,
		Kind:      TypeSlice,
		Fields:    fields,
		FieldMap:  fieldMap,
		SlotCount: 2,
		ElemType:  elem,
	}
}

func makeStrTypeInfo() *TypeInfo {
	info := makeSliceTypeInfo("str", "u8")
	info.Kind = TypeStr
	return info
}

func baseTypes() map[string]*TypeInfo {
	return map[string]*TypeInfo{
		"i32":     {Name: "i32", Kind: TypeI32, SlotCount: 1},
		"u8":      {Name: "u8", Kind: TypeU8, SlotCount: 1},
		"ptr":     {Name: "ptr", Kind: TypePtr, SlotCount: 1},
		"str":     makeStrTypeInfo(),
		"actor":   {Name: "actor", Kind: TypeActor, SlotCount: 1},
		"island":  {Name: "island", Kind: TypeIsland, SlotCount: 1},
		"cap.io":  {Name: "cap.io", Kind: TypeCap, SlotCount: 1},
		"cap.mem": {Name: "cap.mem", Kind: TypeCap, SlotCount: 1},
	}
}

func Check(prog *frontend.Program) (*CheckedProgram, error) {
	file := &frontend.FileAST{Module: "", Structs: prog.Structs, Funcs: prog.Funcs}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

type CheckOptions struct {
	RequireMain bool
}

func CheckWorld(world *module.World) (*CheckedProgram, error) {
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

func CheckWorldOpt(world *module.World, opt CheckOptions) (*CheckedProgram, error) {
	if world == nil || len(world.Files) == 0 {
		return nil, fmt.Errorf("no functions found")
	}

	types := baseTypes()

	type structContext struct {
		module  string
		imports map[string]string
		decl    *frontend.StructDecl
	}

	structs := make(map[string]structContext)
	checked := CheckedProgram{
		MainIndex:          -1,
		Types:              types,
		FuncSigs:           make(map[string]FuncSig),
		GlobalsByModule:    make(map[string]map[string]GlobalInfo),
		GlobalDataByModule: make(map[string][][]byte),
	}
	exportedSymbols := make(map[string]string)

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, st := range file.Structs {
			fullName := qualifyName(module, st.Name)
			if st.Name == "i32" || st.Name == "u8" || st.Name == "ptr" || st.Name == "str" || st.Name == "actor" {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(st.At), st.Name)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate struct '%s'", fullName)
			}
			structs[fullName] = structContext{module: module, imports: imports, decl: st}
			checked.Structs = append(checked.Structs, CheckedStruct{Name: fullName, Module: module, Decl: st})
		}
	}

	state := make(map[string]int)
	var buildType func(name string) (*TypeInfo, error)
	buildType = func(name string) (*TypeInfo, error) {
		if info, ok := types[name]; ok {
			return info, nil
		}
		if elem, ok := sliceElemName(name); ok {
			if elem == "" {
				return nil, fmt.Errorf("invalid slice type '%s'", name)
			}
			if isArrayTypeName(elem) {
				return nil, fmt.Errorf("array element types are not supported yet")
			}
			if elem != "i32" && elem != "u8" {
				return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
			}
			info := makeSliceTypeInfo(name, elem)
			types[name] = info
			return info, nil
		}
		if isArrayTypeName(name) {
			return nil, fmt.Errorf("array types are not supported yet")
		}
		ctx, ok := structs[name]
		if !ok {
			return nil, fmt.Errorf("unknown type '%s'", name)
		}
		switch state[name] {
		case 1:
			return nil, fmt.Errorf("%s: recursive struct '%s'", frontend.FormatPos(ctx.decl.At), name)
		case 2:
			if info, ok := types[name]; ok {
				return info, nil
			}
		}
		state[name] = 1

		fieldMap := make(map[string]FieldInfo)
		var fields []FieldInfo
		slotCount := 0
		for i := range ctx.decl.Fields {
			field := &ctx.decl.Fields[i]
			if _, exists := fieldMap[field.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			resolved, err := resolveTypeName(&field.Type, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			field.Type.Name = resolved
			fieldType, err := buildType(resolved)
			if err != nil {
				return nil, err
			}
			info := FieldInfo{
				Name:      field.Name,
				TypeName:  resolved,
				Offset:    slotCount,
				SlotCount: fieldType.SlotCount,
			}
			fieldMap[field.Name] = info
			fields = append(fields, info)
			slotCount += fieldType.SlotCount
		}

		info := &TypeInfo{
			Name:      name,
			Kind:      TypeStruct,
			Fields:    fields,
			FieldMap:  fieldMap,
			SlotCount: slotCount,
		}
		types[name] = info
		state[name] = 2
		return info, nil
	}

	for name := range structs {
		if _, err := buildType(name); err != nil {
			return nil, err
		}
	}

	builtinSigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		if len(file.Globals) == 0 {
			continue
		}

		fnNames := make(map[string]struct{}, len(file.Funcs))
		for _, fn := range file.Funcs {
			fnNames[fn.Name] = struct{}{}
		}

		globals := make(map[string]GlobalInfo, len(file.Globals))
		var dataBlobs [][]byte
		for _, glob := range file.Globals {
			if glob == nil {
				continue
			}
			if _, exists := globals[glob.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate global '%s'", frontend.FormatPos(glob.At), glob.Name)
			}
			if _, exists := fnNames[glob.Name]; exists {
				return nil, fmt.Errorf("%s: global '%s' conflicts with function '%s'", frontend.FormatPos(glob.At), glob.Name, glob.Name)
			}

			resolved, err := resolveTypeName(&glob.Type, module, imports)
			if err != nil {
				return nil, err
			}
			if resolved == "" {
				if glob.Mutable {
					return nil, fmt.Errorf("%s: global var requires an explicit type annotation", frontend.FormatPos(glob.At))
				}
				if glob.Init == nil {
					return nil, fmt.Errorf("%s: global val requires an initializer to infer its type", frontend.FormatPos(glob.At))
				}
				switch init := glob.Init.(type) {
				case *frontend.NumberExpr:
					resolved = "i32"
				case *frontend.UnaryExpr:
					if init.Op != frontend.TokenMinus {
						return nil, fmt.Errorf("%s: unsupported global val initializer", frontend.FormatPos(glob.At))
					}
					if _, ok := init.X.(*frontend.NumberExpr); !ok {
						return nil, fmt.Errorf("%s: unsupported global val initializer", frontend.FormatPos(glob.At))
					}
					resolved = "i32"
				default:
					return nil, fmt.Errorf("%s: unsupported global val initializer (type inference supports only numeric literals)", frontend.FormatPos(glob.At))
				}
			}
			glob.Type.Name = resolved
			if resolved != "i32" && resolved != "ptr" {
				return nil, fmt.Errorf("%s: global '%s' has unsupported type '%s' (allowed: i32, ptr)", frontend.FormatPos(glob.At), glob.Name, resolved)
			}
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return nil, fmt.Errorf("%s: %v", frontend.FormatPos(glob.At), err)
			}

			dataIndex := len(dataBlobs)
			globals[glob.Name] = GlobalInfo{DataIndex: dataIndex, TypeName: resolved, Mutable: glob.Mutable}

			buf := make([]byte, 8)
			if glob.Mutable {
				dataBlobs = append(dataBlobs, buf)
				continue
			}
			if glob.Init == nil {
				return nil, fmt.Errorf("%s: global val '%s' requires an initializer", frontend.FormatPos(glob.At), glob.Name)
			}
			switch resolved {
			case "ptr":
				if !isNullPtrLiteral(glob.Init) {
					return nil, fmt.Errorf("%s: global val '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, 0)
			case "i32":
				v, ok := constI32(glob.Init)
				if !ok {
					return nil, fmt.Errorf("%s: global val '%s' initializer must be an i32 literal", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
			default:
				return nil, fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
			}
			dataBlobs = append(dataBlobs, buf)
		}

		checked.GlobalsByModule[module] = globals
		checked.GlobalDataByModule[module] = dataBlobs
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, fn := range file.Funcs {
			fullName := qualifyName(module, fn.Name)
			if fn.ExportName != "" {
				if fn.ExportName == "core" || strings.HasPrefix(fn.ExportName, "core.") {
					return nil, fmt.Errorf("%s: @export name must not use the 'core.' namespace", frontend.FormatPos(fn.Pos))
				}
				if strings.HasPrefix(fn.ExportName, "__tetra_") && !strings.HasPrefix(module, "__") {
					return nil, fmt.Errorf("%s: @export name '%s' is reserved for internal runtime modules", frontend.FormatPos(fn.Pos), fn.ExportName)
				}
				if other, exists := exportedSymbols[fn.ExportName]; exists {
					return nil, fmt.Errorf("%s: duplicate @export name '%s' (already used by '%s')", frontend.FormatPos(fn.Pos), fn.ExportName, other)
				}
				exportedSymbols[fn.ExportName] = fullName
			}
			if _, exists := builtinSigs[fullName]; exists {
				return nil, fmt.Errorf("%s: cannot redefine builtin '%s'", frontend.FormatPos(fn.Pos), fullName)
			}
			if _, exists := checked.FuncSigs[fullName]; exists {
				return nil, fmt.Errorf("duplicate function '%s'", fullName)
			}
			retName, err := resolveTypeName(&fn.ReturnType, module, imports)
			if err != nil {
				return nil, err
			}
			fn.ReturnType.Name = retName
			retInfo, err := buildType(retName)
			if err != nil {
				return nil, err
			}
			if retInfo.SlotCount > 2 {
				return nil, fmt.Errorf("function '%s' return type too large", fullName)
			}
			paramTypes := make([]string, 0, len(fn.Params))
			paramSlots := 0
			for i := range fn.Params {
				param := &fn.Params[i]
				resolved, err := resolveTypeName(&param.Type, module, imports)
				if err != nil {
					return nil, err
				}
				param.Type.Name = resolved
				info, err := buildType(resolved)
				if err != nil {
					return nil, err
				}
				paramTypes = append(paramTypes, resolved)
				paramSlots += info.SlotCount
			}
			checked.FuncSigs[fullName] = FuncSig{
				ParamTypes:        paramTypes,
				ParamSlots:        paramSlots,
				ReturnType:        retName,
				ReturnSlots:       retInfo.SlotCount,
				ReturnRegionParam: regionNone,
			}
		}
	}

	for name, sig := range builtinSigs {
		checked.FuncSigs[name] = sig
	}

	if len(checked.FuncSigs) == 0 {
		return nil, fmt.Errorf("no functions found")
	}

	funcCount := 0
	for _, file := range world.Files {
		funcCount += len(file.Funcs)
	}
	maxIter := funcCount + 1
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for _, file := range world.Files {
			module := file.Module
			imports, err := collectImportAliases(file)
			if err != nil {
				return nil, err
			}
			globals := checked.GlobalsByModule[module]
			for _, fn := range file.Funcs {
				fullName := qualifyName(module, fn.Name)
				if len(fn.Body) == 0 {
					return nil, fmt.Errorf("function '%s' must have a body", fullName)
				}
				if _, ok := fn.Body[len(fn.Body)-1].(*frontend.ReturnStmt); !ok {
					return nil, fmt.Errorf("function '%s' must end with return", fullName)
				}

				locals := make(map[string]LocalInfo)
				scopeInfo := newScopeInfo()
				slotIndex := 0
				for _, param := range fn.Params {
					if _, exists := locals[param.Name]; exists {
						return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
					}
					info, err := buildType(param.Type.Name)
					if err != nil {
						return nil, err
					}
					locals[param.Name] = LocalInfo{
						Base:      slotIndex,
						SlotCount: info.SlotCount,
						TypeName:  param.Type.Name,
						Mutable:   false,
					}
					scopeInfo.localScopes[param.Name] = regionNone
					slotIndex += info.SlotCount
				}
				if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
					return nil, err
				}
				state := newRegionState(scopeInfo.localScopes, scopeInfo.islandScopes)
				initParamRegions(fn.Params, state, types)
				if err := checkStmts(fn.Body, locals, globals, checked.FuncSigs, types, module, imports, checked.FuncSigs[fullName].ReturnType, state); err != nil {
					return nil, err
				}
				newReturnParam := regionNone
				if state.returnRegionSet && state.returnRegion < regionNone {
					idx, ok := state.paramRegionIndex[state.returnRegion]
					if !ok {
						return nil, fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(fn.Pos))
					}
					newReturnParam = idx
				}
				sig := checked.FuncSigs[fullName]
				if sig.ReturnRegionParam != newReturnParam {
					sig.ReturnRegionParam = newReturnParam
					checked.FuncSigs[fullName] = sig
					changed = true
				}
			}
		}
		if !changed {
			break
		}
		if iter == maxIter-1 {
			return nil, fmt.Errorf("region inference did not converge")
		}
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		globals := checked.GlobalsByModule[module]
		for _, fn := range file.Funcs {
			fullName := qualifyName(module, fn.Name)
			if fn.Name == "main" {
				if module != world.EntryModule {
					return nil, fmt.Errorf("%s: main must be in entry module", frontend.FormatPos(fn.Pos))
				}
				if len(fn.Params) != 0 {
					return nil, fmt.Errorf("%s: main must not have parameters", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].ReturnType != "i32" {
					return nil, fmt.Errorf("%s: main must return i32", frontend.FormatPos(fn.Pos))
				}
				checked.MainIndex = len(checked.Funcs)
				checked.MainName = fullName
			}
			locals := make(map[string]LocalInfo)
			scopeInfo := newScopeInfo()
			slotIndex := 0
			for _, param := range fn.Params {
				if _, exists := locals[param.Name]; exists {
					return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
				}
				info, err := buildType(param.Type.Name)
				if err != nil {
					return nil, err
				}
				locals[param.Name] = LocalInfo{
					Base:      slotIndex,
					SlotCount: info.SlotCount,
					TypeName:  param.Type.Name,
					Mutable:   false,
				}
				scopeInfo.localScopes[param.Name] = regionNone
				slotIndex += info.SlotCount
			}
			if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
				return nil, err
			}
			checked.Funcs = append(checked.Funcs, CheckedFunc{
				Name:        fullName,
				Module:      module,
				Decl:        fn,
				Locals:      locals,
				LocalSlots:  slotIndex,
				ParamSlots:  checked.FuncSigs[fullName].ParamSlots,
				ReturnType:  checked.FuncSigs[fullName].ReturnType,
				ReturnSlots: checked.FuncSigs[fullName].ReturnSlots,
			})
		}
	}

	if checked.MainIndex == -1 {
		if opt.RequireMain {
			return nil, fmt.Errorf("missing main")
		}
	}

	return &checked, nil
}

func collectImportAliases(file *frontend.FileAST) (map[string]string, error) {
	aliases := make(map[string]string)
	for _, imp := range file.Imports {
		if imp.Alias == "" {
			return nil, fmt.Errorf("%s: import alias required", frontend.FormatPos(imp.At))
		}
		if _, exists := aliases[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), imp.Alias)
		}
		aliases[imp.Alias] = imp.Path
	}
	return aliases, nil
}

func qualifyName(module, name string) string {
	if module == "" {
		return name
	}
	return module + "." + name
}

func resolveTypeName(ref *frontend.TypeRef, module string, imports map[string]string) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing slice element type", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return "", fmt.Errorf("%s: missing array element type", frontend.FormatPos(ref.At))
		}
		if ref.Len < 0 {
			return "", fmt.Errorf("%s: invalid array length", frontend.FormatPos(ref.At))
		}
		elem, err := resolveTypeName(ref.Elem, module, imports)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%d]%s", ref.Len, elem), nil
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", fmt.Errorf("%s: missing type name", frontend.FormatPos(ref.At))
		}
		if ref.Name == "i32" || ref.Name == "u8" || ref.Name == "ptr" || ref.Name == "str" || ref.Name == "island" || ref.Name == "cap.io" || ref.Name == "cap.mem" || ref.Name == "actor" {
			return ref.Name, nil
		}
		parts := strings.Split(ref.Name, ".")
		if len(parts) == 1 {
			return qualifyName(module, ref.Name), nil
		}
		if target, ok := imports[parts[0]]; ok {
			if len(parts) != 2 {
				return "", fmt.Errorf("%s: expected '%s.<type>'", frontend.FormatPos(ref.At), parts[0])
			}
			return target + "." + parts[1], nil
		}
		return ref.Name, nil
	default:
		return "", fmt.Errorf("%s: unsupported type", frontend.FormatPos(ref.At))
	}
}

func resolveCallName(name string, module string, imports map[string]string, pos frontend.Position) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return qualifyName(module, name), nil
	}
	if target, ok := imports[parts[0]]; ok {
		if len(parts) != 2 {
			return "", fmt.Errorf("%s: expected '%s.<func>'", frontend.FormatPos(pos), parts[0])
		}
		return target + "." + parts[1], nil
	}
	modPath := strings.Join(parts[:len(parts)-1], ".")
	return modPath + "." + parts[len(parts)-1], nil
}

const (
	regionNone       = -1
	regionUnknown    = -2
	regionParamStart = -3
)

type scopeInfo struct {
	localScopes  map[string]int
	islandScopes map[string]int
	scopeStack   []int
	nextScopeID  int
}

func newScopeInfo() *scopeInfo {
	return &scopeInfo{
		localScopes:  make(map[string]int),
		islandScopes: make(map[string]int),
	}
}

func (s *scopeInfo) currentScopeID() int {
	if len(s.scopeStack) == 0 {
		return regionNone
	}
	return s.scopeStack[len(s.scopeStack)-1]
}

func (s *scopeInfo) enterScope() int {
	id := s.nextScopeID
	s.nextScopeID++
	s.scopeStack = append(s.scopeStack, id)
	return id
}

func (s *scopeInfo) exitScope() {
	if len(s.scopeStack) == 0 {
		return
	}
	s.scopeStack = s.scopeStack[:len(s.scopeStack)-1]
}

type regionState struct {
	localScopes      map[string]int
	islandScopes     map[string]int
	islandNameByID   map[int]string
	regionVars       map[string]int
	paramRegionIndex map[int]int
	paramNames       []string
	unknownVars      map[string]bool
	unknownConflicts map[string]regionConflict
	activeScopes     []int
	activeIndex      map[int]int
	unsafeDepth      int
	returnRegion     int
	returnRegionSet  bool
}

func newRegionState(localScopes map[string]int, islandScopes map[string]int) *regionState {
	islandNameByID := make(map[int]string, len(islandScopes))
	for name, id := range islandScopes {
		islandNameByID[id] = name
	}
	return &regionState{
		localScopes:      localScopes,
		islandScopes:     islandScopes,
		islandNameByID:   islandNameByID,
		regionVars:       make(map[string]int),
		paramRegionIndex: make(map[int]int),
		unknownConflicts: make(map[string]regionConflict),
		unknownVars:      make(map[string]bool),
		activeIndex:      make(map[int]int),
	}
}

type regionConflict struct {
	leftLabel  string
	leftRegion int

	rightLabel  string
	rightRegion int
}

func (s *regionState) enterIsland(name string) error {
	id, ok := s.islandScopes[name]
	if !ok {
		return fmt.Errorf("unknown island scope '%s'", name)
	}
	s.activeScopes = append(s.activeScopes, id)
	s.activeIndex[id] = len(s.activeScopes) - 1
	s.regionVars[name] = id
	return nil
}

func (s *regionState) exitIsland() {
	if len(s.activeScopes) == 0 {
		return
	}
	id := s.activeScopes[len(s.activeScopes)-1]
	delete(s.activeIndex, id)
	s.activeScopes = s.activeScopes[:len(s.activeScopes)-1]
}

func (s *regionState) isScopeActive(id int) bool {
	if id < 0 {
		return true
	}
	_, ok := s.activeIndex[id]
	return ok
}

func (s *regionState) scopeIndex(id int) (int, bool) {
	idx, ok := s.activeIndex[id]
	return idx, ok
}

func (s *regionState) isScopeWithin(targetID, regionID int) bool {
	if regionID < 0 {
		return true
	}
	if targetID < 0 {
		return false
	}
	regionIdx, ok := s.scopeIndex(regionID)
	if !ok {
		return false
	}
	targetIdx, ok := s.scopeIndex(targetID)
	if !ok {
		return false
	}
	return targetIdx >= regionIdx
}

func copyRegionVars(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeRegionVars(a, b map[string]int) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			vb = regionNone
		}
		if va == vb {
			if va != regionNone {
				merged[k] = va
			}
			continue
		}
		merged[k] = regionUnknown
	}
	for k, vb := range b {
		if _, ok := a[k]; ok {
			continue
		}
		if vb != regionNone {
			merged[k] = regionUnknown
		}
	}
	return merged
}

func joinRegion(a, b int) int {
	if a == regionNone {
		return b
	}
	if b == regionNone {
		return a
	}
	if a == b {
		return a
	}
	return regionUnknown
}

func markUnknownRegions(state *regionState) {
	if state == nil {
		return
	}
	for name := range state.unknownVars {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownVars, name)
		}
	}
	for name := range state.unknownConflicts {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownConflicts, name)
		}
	}
	for name, regionID := range state.regionVars {
		if regionID == regionUnknown {
			state.unknownVars[name] = true
			continue
		}
		delete(state.unknownVars, name)
		delete(state.unknownConflicts, name)
	}
}

func initParamRegions(params []frontend.ParamDecl, state *regionState, types map[string]*TypeInfo) {
	if state != nil && (state.paramNames == nil || len(state.paramNames) != len(params)) {
		state.paramNames = make([]string, len(params))
		for i := range params {
			state.paramNames[i] = params[i].Name
		}
	}
	next := regionParamStart
	for i := range params {
		param := params[i]
		if typeMayContainRegion(param.Type.Name, types) {
			state.regionVars[param.Name] = next
			state.paramRegionIndex[next] = i
			next--
		}
	}
}

func formatRegionID(state *regionState, regionID int) string {
	switch {
	case regionID == regionNone:
		return "none"
	case regionID == regionUnknown:
		return "unknown"
	case regionID >= 0:
		if state != nil {
			if name, ok := state.islandNameByID[regionID]; ok && name != "" {
				return fmt.Sprintf("isl#%d(%s)", regionID, name)
			}
		}
		return fmt.Sprintf("isl#%d", regionID)
	default:
		if state != nil {
			if idx, ok := state.paramRegionIndex[regionID]; ok {
				if idx >= 0 && idx < len(state.paramNames) {
					return fmt.Sprintf("param#%d(%s)", idx, state.paramNames[idx])
				}
				return fmt.Sprintf("param#%d", idx)
			}
		}
		return fmt.Sprintf("param(%d)", regionID)
	}
}

func formatScopeID(state *regionState, scopeID int) string {
	if scopeID == regionNone {
		return "root"
	}
	if scopeID == regionUnknown {
		return "unknown"
	}
	if state != nil {
		if name, ok := state.islandNameByID[scopeID]; ok && name != "" {
			return fmt.Sprintf("scope#%d(%s)", scopeID, name)
		}
	}
	return fmt.Sprintf("scope#%d", scopeID)
}

func recordMergeConflicts(state *regionState, leftVars, rightVars map[string]int, leftLabel, rightLabel string) {
	if state == nil {
		return
	}
	for name, left := range leftVars {
		right, ok := rightVars[name]
		if !ok {
			right = regionNone
		}
		if left == right {
			continue
		}
		if left == regionNone && right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  left,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
	for name, right := range rightVars {
		if _, ok := leftVars[name]; ok {
			continue
		}
		if right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  regionNone,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
}

func (s *regionState) enterUnsafe() {
	s.unsafeDepth++
}

func (s *regionState) exitUnsafe() {
	if s.unsafeDepth > 0 {
		s.unsafeDepth--
	}
}

func (s *regionState) inUnsafe() bool {
	return s.unsafeDepth > 0
}

func (s *regionState) recordReturnRegion(regionID int, pos frontend.Position) error {
	if regionID == regionUnknown {
		return fmt.Errorf("%s: ambiguous region for return", frontend.FormatPos(pos))
	}
	if regionID >= 0 {
		return fmt.Errorf("%s: return from scoped island is not allowed", frontend.FormatPos(pos))
	}
	if !s.returnRegionSet {
		s.returnRegion = regionID
		s.returnRegionSet = true
		return nil
	}
	if s.returnRegion != regionID {
		return fmt.Errorf(
			"%s: return mixes values from different regions (first: %s, now: %s)",
			frontend.FormatPos(pos),
			formatRegionID(s, s.returnRegion),
			formatRegionID(s, regionID),
		)
	}
	return nil
}

func typeMayContainRegion(typeName string, types map[string]*TypeInfo) bool {
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeSlice:
		return true
	case TypeIsland:
		return true
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainRegion(field.TypeName, types) {
				return true
			}
		}
		return false
	case TypeArray:
		return typeMayContainRegion(info.ElemType, types)
	default:
		return false
	}
}

func localScopeID(name string, state *regionState) int {
	if state == nil {
		return regionNone
	}
	if id, ok := state.localScopes[name]; ok {
		return id
	}
	return regionNone
}

func checkLocalScope(name string, state *regionState, pos frontend.Position) error {
	scopeID := localScopeID(name, state)
	if scopeID == regionNone {
		return nil
	}
	if !state.isScopeActive(scopeID) {
		return fmt.Errorf("%s: identifier '%s' is out of scope", frontend.FormatPos(pos), name)
	}
	return nil
}

func checkStmts(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	state *regionState,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			tname, _, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !isPrintableType(tname, types) {
				return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
			}
		case *frontend.FreeStmt:
			tname, _, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if tname != "island" {
				return fmt.Errorf("%s: free expects island, got '%s'", frontend.FormatPos(s.At), tname)
			}
			if !s.Implicit && !state.inUnsafe() {
				return fmt.Errorf("%s: free is only allowed in unsafe blocks", frontend.FormatPos(s.At))
			}
		case *frontend.ReturnStmt:
			tname, regionID, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if err := state.recordReturnRegion(regionID, s.At); err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(returnType, tname, s.Value) {
				return fmt.Errorf("%s: return type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), returnType, tname)
			}
		case *frontend.IslandStmt:
			sizeType, _, err := checkExpr(s.Size, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !isInt32Like(sizeType) {
				return fmt.Errorf("%s: island size must be i32/u8", frontend.FormatPos(s.At))
			}
			if err := state.enterIsland(s.Name); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, state); err != nil {
				return err
			}
			state.exitIsland()
		case *frontend.LetStmt:
			resolved, err := resolveTypeName(&s.Type, module, imports)
			if err != nil {
				return err
			}
			s.Type.Name = resolved
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			valType, valRegion, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(resolved, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), resolved, valType)
			}
			if valRegion >= 0 {
				scopeID := localScopeID(s.Name, state)
				if !state.isScopeWithin(scopeID, valRegion) {
					return fmt.Errorf(
						"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
						frontend.FormatPos(s.At),
						formatRegionID(state, valRegion),
						formatScopeID(state, scopeID),
					)
				}
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else if valRegion < regionNone {
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else {
				delete(state.regionVars, s.Name)
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			}
		case *frontend.AssignStmt:
			if idx, ok := s.Target.(*frontend.IndexExpr); ok {
				indexType, _, err := checkExpr(idx.Index, locals, globals, funcs, types, module, imports, state)
				if err != nil {
					return err
				}
				if !isInt32Like(indexType) {
					return fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(idx.At))
				}
				if _, _, err := checkExpr(idx.Base, locals, globals, funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if g, ok := globals[id.Name]; ok {
					if !g.Mutable {
						return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), id.Name)
					}
					valType, _, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
					if err != nil {
						return err
					}
					if !typesCompatibleWithNullPtr(g.TypeName, valType, s.Value) {
						return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), g.TypeName, valType)
					}
					continue
				}
			}
			targetInfo, targetType, err := resolveAssignTarget(s.Target, locals, types)
			if err != nil {
				return err
			}
			if err := checkLocalScope(targetInfo.Name, state, s.At); err != nil {
				return err
			}
			if !targetInfo.Mutable {
				return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), targetInfo.Name)
			}
			valType, valRegion, err := checkExpr(s.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(targetType, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), targetType, valType)
			}
			if _, ok := s.Target.(*frontend.IndexExpr); !ok {
				if valRegion >= 0 {
					scopeID := localScopeID(targetInfo.Name, state)
					if !state.isScopeWithin(scopeID, valRegion) {
						return fmt.Errorf(
							"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
							frontend.FormatPos(s.At),
							formatRegionID(state, valRegion),
							formatScopeID(state, scopeID),
						)
					}
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else if valRegion < regionNone {
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else {
					delete(state.regionVars, targetInfo.Name)
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				}
			}
		case *frontend.IfStmt:
			condType, _, err := checkExpr(s.Cond, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !isInt32Like(condType) {
				return fmt.Errorf("%s: condition must be i32/u8", frontend.FormatPos(s.At))
			}
			before := copyRegionVars(state.regionVars)
			state.regionVars = copyRegionVars(before)
			if err := checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, state); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			var elseVars map[string]int
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				if err := checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, state); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
			} else {
				elseVars = before
			}
			state.regionVars = mergeRegionVars(thenVars, elseVars)
			recordMergeConflicts(state, thenVars, elseVars, "then", "else")
			markUnknownRegions(state)
		case *frontend.WhileStmt:
			condType, _, err := checkExpr(s.Cond, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return err
			}
			if !isInt32Like(condType) {
				return fmt.Errorf("%s: condition must be i32/u8", frontend.FormatPos(s.At))
			}
			before := copyRegionVars(state.regionVars)
			state.regionVars = copyRegionVars(before)
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, state); err != nil {
				return err
			}
			bodyVars := copyRegionVars(state.regionVars)
			state.regionVars = mergeRegionVars(before, bodyVars)
			recordMergeConflicts(state, before, bodyVars, "before", "body")
			markUnknownRegions(state)
		case *frontend.UnsafeStmt:
			state.enterUnsafe()
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, state); err != nil {
				return err
			}
			state.exitUnsafe()
		default:
			return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
		}
	}
	return nil
}

func collectLocals(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			resolved := ""
			if s.Type.Kind == frontend.TypeRefNamed && s.Type.Name == "" {
				inferred, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer type for '%s': %v", frontend.FormatPos(s.At), s.Name, err)
				}
				resolved = inferred
				s.Type = frontend.TypeRef{At: s.At, Kind: frontend.TypeRefNamed, Name: inferred}
			} else {
				var err error
				resolved, err = resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				s.Type.Name = resolved
			}
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  resolved,
				Mutable:   s.Mutable,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
		case *frontend.IslandStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			islandInfo, err := ensureTypeInfo("island", types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
				scopes.localScopes[s.Name] = scopeID
				scopes.islandScopes[s.Name] = scopeID
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: islandInfo.SlotCount,
				TypeName:  "island",
				Mutable:   false,
			}
			*slotIndex += islandInfo.SlotCount
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		case *frontend.IfStmt:
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if len(s.Else) > 0 {
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
		case *frontend.WhileStmt:
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.UnsafeStmt:
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	}
	return nil
}

func inferExprTypeForDecl(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		if info, ok := locals[e.Name]; ok {
			if info.TypeName == "" {
				return "", fmt.Errorf("depends on '%s' which has no type annotation", e.Name)
			}
			return info.TypeName, nil
		}
		if g, ok := globals[e.Name]; ok {
			return g.TypeName, nil
		}
		return "", fmt.Errorf("unknown identifier '%s'", e.Name)
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return "", fmt.Errorf("unsupported unary operator")
		}
		return "i32", nil
	case *frontend.BinaryExpr:
		if e.Op != frontend.TokenPlus && e.Op != frontend.TokenMinus && e.Op != frontend.TokenEqEq && e.Op != frontend.TokenLess {
			return "", fmt.Errorf("unsupported binary operator")
		}
		return "i32", nil
	case *frontend.FieldAccessExpr:
		_, targetType, err := ResolveFieldAccessType(e, locals, types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		baseType, err := inferExprTypeForDecl(e.Base, locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return "", err
		}
		switch info.Kind {
		case TypeStr:
			return "u8", nil
		case TypeSlice:
			return info.ElemType, nil
		default:
			return "", fmt.Errorf("cannot index '%s'", baseType)
		}
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		return resolved, nil
	case *frontend.CallExpr:
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
			resolved = builtin
		} else {
			name, err := resolveCallName(e.Name, module, imports, e.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		return sig.ReturnType, nil
	default:
		return "", fmt.Errorf("unsupported expression for type inference")
	}
}

func checkExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) (string, int, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", regionNone, nil
	case *frontend.StringLitExpr:
		return "str", regionNone, nil
	case *frontend.IdentExpr:
		if err := checkLocalScope(e.Name, state, e.At); err != nil {
			return "", regionNone, err
		}
		info, ok := locals[e.Name]
		if !ok {
			if g, ok := globals[e.Name]; ok {
				return g.TypeName, regionNone, nil
			}
			return "", regionNone, fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(e.At), e.Name)
		}
		if regionID, ok := state.regionVars[e.Name]; ok {
			if regionID == regionUnknown {
				if state.unknownVars[e.Name] {
					if conflict, ok := state.unknownConflicts[e.Name]; ok {
						return "", regionNone, fmt.Errorf(
							"%s: ambiguous region for '%s' after control-flow merge (%s: %s, %s: %s); hint: assign to a fresh variable in each branch and use it after the merge",
							frontend.FormatPos(e.At),
							e.Name,
							conflict.leftLabel,
							formatRegionID(state, conflict.leftRegion),
							conflict.rightLabel,
							formatRegionID(state, conflict.rightRegion),
						)
					}
					return "", regionNone, fmt.Errorf(
						"%s: ambiguous region for '%s' after control-flow merge; hint: reassign it to a single region before use",
						frontend.FormatPos(e.At),
						e.Name,
					)
				}
				return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s'", frontend.FormatPos(e.At), e.Name)
			}
			if !state.isScopeActive(regionID) {
				return "", regionNone, fmt.Errorf("%s: slice from scoped island is out of scope", frontend.FormatPos(e.At))
			}
			return info.TypeName, regionID, nil
		}
		return info.TypeName, regionNone, nil
	case *frontend.FieldAccessExpr:
		targetInfo, targetType, err := ResolveFieldAccessType(e, locals, types)
		if err != nil {
			return "", regionNone, err
		}
		baseType, baseRegion, err := checkExpr(e.Base, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		if baseType == "" {
			return "", regionNone, fmt.Errorf("%s: invalid field access base", frontend.FormatPos(e.At))
		}
		if err := checkLocalScope(targetInfo.Name, state, e.At); err != nil {
			return "", regionNone, err
		}
		if typeMayContainRegion(targetType, types) && baseRegion != regionNone {
			return targetType, baseRegion, nil
		}
		return targetType, regionNone, nil
	case *frontend.IndexExpr:
		baseType, _, err := checkExpr(e.Base, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		indexType, _, err := checkExpr(e.Index, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		if !isInt32Like(indexType) {
			return "", regionNone, fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(e.At))
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return "", regionNone, err
		}
		switch info.Kind {
		case TypeStr:
			return "u8", regionNone, nil
		case TypeSlice:
			return info.ElemType, regionNone, nil
		default:
			return "", regionNone, fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(e.At), baseType)
		}
	case *frontend.CallExpr:
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
			resolved = builtin
		} else {
			var err error
			resolved, err = resolveCallName(e.Name, module, imports, e.At)
			if err != nil {
				return "", regionNone, err
			}
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), resolved)
		}
		if (resolved == "core.actor_dispatch" || resolved == "core.actor_main_entry_id") && !strings.HasPrefix(module, "__") {
			return "", regionNone, fmt.Errorf("%s: '%s' is reserved for internal runtime modules", frontend.FormatPos(e.At), resolved)
		}
		if len(e.Args) != len(sig.ParamTypes) {
			return "", regionNone, fmt.Errorf("%s: wrong argument count for '%s'", frontend.FormatPos(e.At), resolved)
		}
		argRegions := make([]int, len(e.Args))
		for i, arg := range e.Args {
			argType, argRegion, err := checkExpr(arg, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return "", regionNone, err
			}
			if !typesCompatibleWithNullPtr(sig.ParamTypes[i], argType, arg) {
				return "", regionNone, fmt.Errorf("%s: type mismatch for '%s' arg %d", frontend.FormatPos(arg.Pos()), resolved, i+1)
			}
			argRegions[i] = argRegion
		}
		if resolved == "core.spawn" {
			if len(e.Args) != 1 {
				return "", regionNone, fmt.Errorf("%s: spawn expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return "", regionNone, fmt.Errorf("%s: spawn expects a string literal", frontend.FormatPos(e.At))
			}
			raw := string(lit.Value)
			if raw == "" {
				return "", regionNone, fmt.Errorf("%s: spawn expects a non-empty name", frontend.FormatPos(e.At))
			}
			target, err := resolveCallName(raw, module, imports, e.At)
			if err != nil {
				return "", regionNone, err
			}
			if strings.HasPrefix(target, "core.") {
				return "", regionNone, fmt.Errorf("%s: spawn target must be a user function, got '%s'", frontend.FormatPos(e.At), target)
			}
			targetSig, ok := funcs[target]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
			}
			if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
				return "", regionNone, fmt.Errorf("%s: spawn target must have shape fun %s(): i32", frontend.FormatPos(e.At), target)
			}
			lit.Value = []byte(target)
		}
		if resolved == "core.sym_addr" {
			if len(e.Args) != 1 {
				return "", regionNone, fmt.Errorf("%s: sym_addr expects 1 argument", frontend.FormatPos(e.At))
			}
			lit, ok := e.Args[0].(*frontend.StringLitExpr)
			if !ok {
				return "", regionNone, fmt.Errorf("%s: sym_addr expects a string literal", frontend.FormatPos(e.At))
			}
			if len(lit.Value) == 0 {
				return "", regionNone, fmt.Errorf("%s: sym_addr expects a non-empty symbol name", frontend.FormatPos(e.At))
			}
		}
		if (resolved == "core.island_make_u8" || resolved == "core.island_make_i32") && len(argRegions) > 0 && argRegions[0] == regionUnknown {
			return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s' argument", frontend.FormatPos(e.At), resolved)
		}
		if builtinNeedsUnsafe(resolved, argRegions) && !state.inUnsafe() {
			return "", regionNone, fmt.Errorf("%s: '%s' is only allowed in unsafe blocks", frontend.FormatPos(e.At), resolved)
		}
		e.Name = resolved
		regionID := regionNone
		if sig.ReturnRegionParam >= 0 {
			if sig.ReturnRegionParam >= len(argRegions) {
				return "", regionNone, fmt.Errorf("%s: invalid region signature for '%s'", frontend.FormatPos(e.At), resolved)
			}
			regionID = argRegions[sig.ReturnRegionParam]
			if regionID == regionUnknown {
				return "", regionNone, fmt.Errorf("%s: ambiguous region for '%s' return", frontend.FormatPos(e.At), resolved)
			}
		}
		return sig.ReturnType, regionID, nil
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", regionNone, err
		}
		e.Type.Name = resolved
		info, err := ensureTypeInfo(resolved, types)
		if err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		if info.Kind != TypeStruct {
			return "", regionNone, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(e.At), resolved)
		}
		seen := make(map[string]frontend.StructFieldInit, len(e.Fields))
		for _, field := range e.Fields {
			if _, exists := info.FieldMap[field.Name]; !exists {
				return "", regionNone, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			if _, exists := seen[field.Name]; exists {
				return "", regionNone, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			seen[field.Name] = field
		}
		structRegion := regionNone
		for _, field := range info.Fields {
			init, ok := seen[field.Name]
			if !ok {
				return "", regionNone, fmt.Errorf("%s: missing field '%s'", frontend.FormatPos(e.At), field.Name)
			}
			valType, valRegion, err := checkExpr(init.Value, locals, globals, funcs, types, module, imports, state)
			if err != nil {
				return "", regionNone, err
			}
			if !typesCompatibleWithNullPtr(field.TypeName, valType, init.Value) {
				return "", regionNone, fmt.Errorf("%s: type mismatch for field '%s'", frontend.FormatPos(init.At), field.Name)
			}
			structRegion = joinRegion(structRegion, valRegion)
			if structRegion == regionUnknown {
				return "", regionNone, fmt.Errorf("%s: struct literal mixes values from different regions", frontend.FormatPos(init.At))
			}
		}
		return resolved, structRegion, nil
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return "", regionNone, fmt.Errorf("%s: unsupported unary operator", frontend.FormatPos(e.At))
		}
		xtype, _, err := checkExpr(e.X, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		if !isInt32Like(xtype) {
			return "", regionNone, fmt.Errorf("%s: unary '-' expects i32/u8", frontend.FormatPos(e.At))
		}
		return "i32", regionNone, nil
	case *frontend.BinaryExpr:
		if e.Op != frontend.TokenPlus && e.Op != frontend.TokenMinus && e.Op != frontend.TokenEqEq && e.Op != frontend.TokenLess {
			return "", regionNone, fmt.Errorf("%s: unsupported binary operator", frontend.FormatPos(e.At))
		}
		ltype, _, err := checkExpr(e.Left, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		rtype, _, err := checkExpr(e.Right, locals, globals, funcs, types, module, imports, state)
		if err != nil {
			return "", regionNone, err
		}
		if !isInt32Like(ltype) || !isInt32Like(rtype) {
			return "", regionNone, fmt.Errorf("%s: binary operators require i32/u8", frontend.FormatPos(e.At))
		}
		return "i32", regionNone, nil
	default:
		return "", regionNone, fmt.Errorf("%s: unsupported expression", frontend.FormatPos(expr.Pos()))
	}
}

type assignTargetInfo struct {
	Name     string
	Mutable  bool
	TypeName string
	Offset   int
}

func resolveAssignTarget(expr frontend.Expr, locals map[string]LocalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		baseName, fields, pos, ok := splitFieldPath(idx.Base)
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
		}
		baseInfo, ok := locals[baseName]
		if !ok {
			return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
		}
		if _, err := ensureTypeInfo(baseInfo.TypeName, types); err != nil {
			return assignTargetInfo{}, "", err
		}
		baseType, _, _, err := resolveFieldChain(baseInfo.TypeName, baseInfo.Base, fields, types, pos)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return assignTargetInfo{}, "", err
		}
		if info.Kind == TypeStr {
			return assignTargetInfo{}, "", fmt.Errorf("%s: cannot assign into str", frontend.FormatPos(pos))
		}
		if info.Kind != TypeSlice {
			return assignTargetInfo{}, "", fmt.Errorf("%s: cannot index '%s'", frontend.FormatPos(pos), baseType)
		}
		return assignTargetInfo{Name: baseName, Mutable: baseInfo.Mutable, TypeName: info.ElemType}, info.ElemType, nil
	}

	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid assignment target", frontend.FormatPos(pos))
	}
	info, ok := locals[baseName]
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{Name: baseName, Mutable: info.Mutable, TypeName: targetType, Offset: offset}, targetType, nil
}

func ResolveFieldAccessType(expr frontend.Expr, locals map[string]LocalInfo, types map[string]*TypeInfo) (assignTargetInfo, string, error) {
	baseName, fields, pos, ok := splitFieldPath(expr)
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: invalid field access", frontend.FormatPos(pos))
	}
	info, ok := locals[baseName]
	if !ok {
		return assignTargetInfo{}, "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), baseName)
	}
	if _, err := ensureTypeInfo(info.TypeName, types); err != nil {
		return assignTargetInfo{}, "", err
	}
	targetType, _, offset, err := resolveFieldChain(info.TypeName, info.Base, fields, types, pos)
	if err != nil {
		return assignTargetInfo{}, "", err
	}
	return assignTargetInfo{Name: baseName, Mutable: info.Mutable, TypeName: targetType, Offset: offset}, targetType, nil
}

func splitFieldPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := splitFieldPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func resolveFieldChain(typeName string, baseOffset int, fields []string, types map[string]*TypeInfo, pos frontend.Position) (string, int, int, error) {
	offset := baseOffset
	current := typeName
	for _, field := range fields {
		info, ok := types[current]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
		}
		if info.Kind != TypeStruct && info.Kind != TypeSlice && info.Kind != TypeStr {
			return "", 0, 0, fmt.Errorf("%s: '%s' is not a struct", frontend.FormatPos(pos), current)
		}
		fieldInfo, ok := info.FieldMap[field]
		if !ok {
			return "", 0, 0, fmt.Errorf("%s: unknown field '%s'", frontend.FormatPos(pos), field)
		}
		offset += fieldInfo.Offset
		current = fieldInfo.TypeName
	}
	info, ok := types[current]
	if !ok {
		return "", 0, 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), current)
	}
	return current, info.SlotCount, offset, nil
}

func ensureTypeInfo(name string, types map[string]*TypeInfo) (*TypeInfo, error) {
	if info, ok := types[name]; ok {
		return info, nil
	}
	if elem, ok := sliceElemName(name); ok {
		if elem != "i32" && elem != "u8" {
			return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
		}
		info := makeSliceTypeInfo(name, elem)
		types[name] = info
		return info, nil
	}
	if isArrayTypeName(name) {
		return nil, fmt.Errorf("array types are not supported yet")
	}
	return nil, fmt.Errorf("unknown type '%s'", name)
}

func typesCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if expected == "u8" && actual == "i32" {
		return true
	}
	if expected == "i32" && actual == "u8" {
		return true
	}
	return false
}

func typesCompatibleWithNullPtr(expected, actual string, expr frontend.Expr) bool {
	if typesCompatible(expected, actual) {
		return true
	}
	if expected == "ptr" && actual == "i32" && isNullPtrLiteral(expr) {
		return true
	}
	return false
}

func isNullPtrLiteral(expr frontend.Expr) bool {
	n, ok := expr.(*frontend.NumberExpr)
	return ok && n.Value == 0
}

func constI32(expr frontend.Expr) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := e.X.(*frontend.NumberExpr)
		if !ok {
			return 0, false
		}
		return -v.Value, true
	default:
		return 0, false
	}
}

func isInt32Like(name string) bool {
	return name == "i32" || name == "u8"
}

func isPrintableType(name string, types map[string]*TypeInfo) bool {
	info, err := ensureTypeInfo(name, types)
	if err != nil {
		return false
	}
	if info.Kind == TypeStr {
		return true
	}
	if info.Kind == TypeSlice && info.ElemType == "u8" {
		return true
	}
	return false
}

func builtinFuncSigs(types map[string]*TypeInfo) (map[string]FuncSig, error) {
	_, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}
	actorInfo, err := ensureTypeInfo("actor", types)
	if err != nil {
		return nil, err
	}
	ptrInfo, err := ensureTypeInfo("ptr", types)
	if err != nil {
		return nil, err
	}
	sliceU8, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	sliceI32, err := ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}

	islandInfo, err := ensureTypeInfo("island", types)
	if err != nil {
		return nil, err
	}
	capIO, err := ensureTypeInfo("cap.io", types)
	if err != nil {
		return nil, err
	}
	capMem, err := ensureTypeInfo("cap.mem", types)
	if err != nil {
		return nil, err
	}

	return map[string]FuncSig{
		"core.alloc_bytes":         {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u8":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.make_i32":            {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.island_new":          {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "island", ReturnSlots: islandInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.island_make_u8":      {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.island_make_i32":     {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.cap_io":              {ParamTypes: nil, ParamSlots: 0, ReturnType: capIO.Name, ReturnSlots: capIO.SlotCount, ReturnRegionParam: regionNone},
		"core.cap_mem":             {ParamTypes: nil, ParamSlots: 0, ReturnType: capMem.Name, ReturnSlots: capMem.SlotCount, ReturnRegionParam: regionNone},
		"core.load_i32":            {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_i32":           {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_u8":             {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_u8":            {ParamTypes: []string{"ptr", "u8", capMem.Name}, ParamSlots: 3, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_ptr":            {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.store_ptr":           {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ptr_add":             {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.mmio_read_i32":       {ParamTypes: []string{"ptr", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.mmio_write_i32":      {ParamTypes: []string{"ptr", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sym_addr":            {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ctx_switch":          {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_dispatch":      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_main_entry_id": {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.spawn":               {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.send":                {ParamTypes: []string{"actor", "i32"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv":                {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.self":                {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.sender":              {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
	}, nil
}

func builtinNeedsUnsafe(name string, argRegions []int) bool {
	switch name {
	case "core.alloc_bytes", "core.island_new", "core.cap_io", "core.cap_mem",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr",
		"core.ptr_add",
		"core.mmio_read_i32", "core.mmio_write_i32",
		"core.sym_addr", "core.ctx_switch":
		return true
	case "core.island_make_u8", "core.island_make_i32":
		if len(argRegions) == 0 {
			return true
		}
		return argRegions[0] == regionNone
	default:
		return false
	}
}

func ResolveBuiltinAlias(name string) (string, bool) {
	switch name {
	case "alloc_bytes":
		return "core.alloc_bytes", true
	case "make_u8":
		return "core.make_u8", true
	case "make_i32":
		return "core.make_i32", true
	case "island_new":
		return "core.island_new", true
	case "island_make_u8":
		return "core.island_make_u8", true
	case "island_make_i32":
		return "core.island_make_i32", true
	case "load_ptr":
		return "core.load_ptr", true
	case "store_ptr":
		return "core.store_ptr", true
	case "sym_addr":
		return "core.sym_addr", true
	case "ctx_switch":
		return "core.ctx_switch", true
	case "actor_dispatch":
		return "core.actor_dispatch", true
	case "actor_main_entry_id":
		return "core.actor_main_entry_id", true
	case "core.alloc_bytes", "core.make_u8", "core.make_i32",
		"core.island_new", "core.island_make_u8", "core.island_make_i32",
		"core.load_ptr", "core.store_ptr", "core.sym_addr", "core.ctx_switch",
		"core.actor_dispatch", "core.actor_main_entry_id":
		return name, true
	default:
		return "", false
	}
}

func sliceElemName(name string) (string, bool) {
	if strings.HasPrefix(name, "[]") {
		return name[2:], true
	}
	return "", false
}

func isArrayTypeName(name string) bool {
	return strings.HasPrefix(name, "[") && strings.Contains(name, "]")
}
