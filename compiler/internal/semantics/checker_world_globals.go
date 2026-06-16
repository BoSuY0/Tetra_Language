package semantics

import (
	"encoding/binary"
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func collectWorldGlobals(world *module.World, checked *CheckedProgram, types map[string]*TypeInfo) error {
	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		if len(file.Globals) == 0 {
			continue
		}

		fnNames := make(map[string]struct{}, len(file.Funcs))
		for _, fn := range file.Funcs {
			fnNames[fn.Name] = struct{}{}
		}

		globals := make(map[string]GlobalInfo, len(file.Globals))
		constValues := make(map[string]globalConstValue, len(file.Globals))
		var dataBlobs [][]byte
		for _, glob := range file.Globals {
			if glob == nil {
				continue
			}
			if _, exists := globals[glob.Name]; exists {
				return fmt.Errorf("%s: duplicate global '%s'", frontend.FormatPos(glob.At), glob.Name)
			}
			if _, exists := fnNames[glob.Name]; exists {
				return fmt.Errorf("%s: global '%s' conflicts with function '%s'", frontend.FormatPos(glob.At), glob.Name, glob.Name)
			}

			resolved := ""
			functionTypeValue := glob.Type.Kind == frontend.TypeRefFunction
			functionParamTypes := []string(nil)
			functionParamOwnership := []string(nil)
			functionReturnType := ""
			functionReturnOwnership := ""
			functionThrowsType := ""
			functionEffects := []string(nil)
			if glob.Type.Name != "" || glob.Type.Elem != nil || functionTypeValue {
				var err error
				if functionTypeValue {
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(glob.Type, module, imports)
					if err != nil {
						return err
					}
					functionParamOwnership = functionTypeRefParamOwnership(glob.Type)
					functionReturnOwnership = functionTypeRefReturnOwnership(glob.Type)
					functionThrowsType, err = functionTypeRefThrowsType(glob.Type, module, imports)
					if err != nil {
						return err
					}
				}
				resolved, err = resolveTypeName(&glob.Type, module, imports)
				if err != nil {
					return err
				}
			}
			if resolved == "" {
				if glob.Mutable {
					return fmt.Errorf("%s: global var requires an explicit type annotation", frontend.FormatPos(glob.At))
				}
				if glob.Init == nil {
					return fmt.Errorf("%s: global val requires an initializer to infer its type", frontend.FormatPos(glob.At))
				}
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				inferred, ok := inferGlobalConstExprType(glob.Init, constValues)
				if !ok {
					return fmt.Errorf("%s: unsupported global val initializer (type inference supports constant numeric/bool expressions)", frontend.FormatPos(glob.At))
				}
				resolved = inferred
			}
			glob.Type.Name = resolved
			if !isSupportedGlobalType(resolved, types) {
				return fmt.Errorf("%s: global '%s' has unsupported type '%s' (allowed: i32, bool, ptr, str, u8, u16, task.error)", frontend.FormatPos(glob.At), glob.Name, resolved)
			}
			typeInfo, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(glob.At), err)
			}
			if surfaceType, ok := surfaceEphemeralValueType(resolved, types); ok {
				return lifetimeDiagnosticf(glob.At, "surface value '%s' cannot be stored in global '%s'; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType, glob.Name)
			}
			functionValue := ""
			if functionTypeValue {
				switch init := glob.Init.(type) {
				case *frontend.IdentExpr:
					if _, ok := fnNames[init.Name]; !ok {
						return unsupportedFunctionTypedGlobalSameModuleInitializerError(init.At, glob.Name)
					}
					functionValue = qualifyName(module, init.Name)
				case *frontend.FieldAccessExpr:
					resolved, targetSig, ok, err := resolveImportedFunctionGlobalInitializer(init, world, module, imports, types)
					if err != nil {
						return err
					}
					if !ok {
						return unsupportedFunctionTypedGlobalImportedInitializerError(init.At, glob.Name)
					}
					if targetSig.Generic {
						return unsupportedGenericFunctionTypedGlobalInitializerError(init.At, callbackArgumentName(init), glob.Name)
					}
					if err := validateFunctionTypeSymbolSignature(glob.Name, glob.Type, targetSig, module, imports, init.At); err != nil {
						return err
					}
					functionValue = resolved
				case *frontend.ClosureExpr:
					if err := validateFunctionTypeLiteralBinding(glob.Name, glob.Type, init, nil, module, imports); err != nil {
						return err
					}
					functionValue = qualifyName(module, init.Name)
				default:
					return unsupportedFunctionTypedGlobalInitializerSourceError(glob.At, glob.Name)
				}
			}
			stringInit := []byte(nil)
			hasStringInit := false
			if resolved == "str" {
				if glob.Init == nil {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return fmt.Errorf("%s: global %s '%s' initializer must be a string literal", frontend.FormatPos(glob.At), kind, glob.Name)
				}
				lit, ok := glob.Init.(*frontend.StringLitExpr)
				if !ok {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return fmt.Errorf("%s: global %s '%s' initializer must be a string literal", frontend.FormatPos(glob.Init.Pos()), kind, glob.Name)
				}
				hasStringInit = true
				stringInit = append([]byte(nil), lit.Value...)
			}

			dataIndex := len(dataBlobs)
			arrayBackings := collectGlobalArrayBackings(resolved, 0, types, map[string]bool{})
			globals[glob.Name] = GlobalInfo{
				DataIndex:               dataIndex,
				TypeName:                resolved,
				Mutable:                 glob.Mutable,
				Const:                   glob.Const,
				Public:                  declarationIsPublic(file, glob.Public),
				FunctionValue:           functionValue,
				FunctionTypeValue:       functionTypeValue,
				FunctionParamTypes:      functionParamTypes,
				FunctionParamOwnership:  functionParamOwnership,
				FunctionReturnType:      functionReturnType,
				FunctionReturnOwnership: functionReturnOwnership,
				FunctionThrowsType:      functionThrowsType,
				FunctionEffects:         functionEffects,
				HasStringLiteralInit:    hasStringInit,
				StringLiteralInit:       stringInit,
				ArrayBackings:           arrayBackings,
			}

			slots := typeInfo.SlotCount
			if slots <= 0 {
				slots = 1
			}
			slotData := make([][]byte, slots)
			for i := 0; i < slots; i++ {
				slotData[i] = make([]byte, 8)
			}
			buf := slotData[0]
			if glob.Mutable {
				if glob.Init != nil {
					switch resolved {
					case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return fmt.Errorf("%s: global var '%s' initializer must be an i32 constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if err := validateGlobalIntLikeRange(resolved, glob.Name, "var", glob.Init.Pos(), v); err != nil {
							return err
						}
						binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
					case "bool":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstBool(glob.Init, constValues)
						if !ok {
							return fmt.Errorf("%s: global var '%s' initializer must be a bool constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if v {
							binary.LittleEndian.PutUint64(buf, 1)
						} else {
							binary.LittleEndian.PutUint64(buf, 0)
						}
					case "ptr":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return fmt.Errorf("%s: global var '%s' initializer for type ptr must be a constant 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if v != 0 {
							return fmt.Errorf("%s: global var '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						binary.LittleEndian.PutUint64(buf, 0)
					case "str":
						binary.LittleEndian.PutUint64(slotData[0], 0)
						binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
					case "fnptr":
						// Function-typed mutable globals are lazily initialized by lowering
						// because function symbol addresses are not static data constants.
					default:
						if isSupportedSliceGlobalType(resolved) {
							return fmt.Errorf("%s: global var '%s' initializer for type %s must be omitted", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
						}
						if isSupportedOptionalPtrGlobalType(resolved) {
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf("%s: global var '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
						}
						if isSupportedOptionalSliceGlobalType(resolved) {
							if glob.Init == nil {
								break
							}
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf("%s: global var '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
						}
						if isSupportedOptionalAggregateGlobalType(resolved, types) {
							if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
								break
							}
							return fmt.Errorf("%s: global var '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
						}
						if isSupportedZeroedAggregateGlobalType(resolved, types, map[string]bool{}) {
							return fmt.Errorf("%s: global var '%s' initializer for type %s must be omitted", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
						}
						return fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
					}
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			if glob.Init == nil {
				switch resolved {
				case "ptr":
					binary.LittleEndian.PutUint64(buf, 0)
				case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: 0}
				case "bool":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: false}
				case "str":
					binary.LittleEndian.PutUint64(slotData[0], 0)
					binary.LittleEndian.PutUint64(slotData[1], 0)
				default:
					if isSupportedSliceGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalPtrGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalSliceGlobalType(resolved) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedOptionalAggregateGlobalType(resolved, types) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					if isSupportedZeroedAggregateGlobalType(resolved, types, map[string]bool{}) {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			switch resolved {
			case "fnptr":
				// Function-typed globals carry symbol metadata in GlobalInfo; data slots
				// stay zeroed because this MVP only permits immutable direct symbols.
			case "ptr":
				if !isNullPtrLiteral(glob.Init) {
					return fmt.Errorf("%s: global val '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, 0)
			case "i32", "u8", "u16", "c_int", "c_uint", "task.error":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				v, ok := evalGlobalConstI32(glob.Init, constValues)
				if !ok {
					return fmt.Errorf("%s: global val '%s' initializer must be an i32 constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				if err := validateGlobalIntLikeRange(resolved, glob.Name, "val", glob.Init.Pos(), v); err != nil {
					return err
				}
				binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
				constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: v}
			case "bool":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return err
				}
				v, ok := evalGlobalConstBool(glob.Init, constValues)
				if !ok {
					return fmt.Errorf("%s: global val '%s' initializer must be a bool constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				if v {
					binary.LittleEndian.PutUint64(buf, 1)
				} else {
					binary.LittleEndian.PutUint64(buf, 0)
				}
				constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: v}
			case "str":
				binary.LittleEndian.PutUint64(slotData[0], 0)
				binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
			default:
				if isSupportedSliceGlobalType(resolved) {
					return fmt.Errorf("%s: global val '%s' initializer for type %s must be omitted", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
				}
				if isSupportedOptionalPtrGlobalType(resolved) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf("%s: global val '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
				}
				if isSupportedOptionalSliceGlobalType(resolved) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf("%s: global val '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
				}
				if isSupportedOptionalAggregateGlobalType(resolved, types) {
					if _, ok := glob.Init.(*frontend.NoneLitExpr); ok {
						dataBlobs = append(dataBlobs, slotData...)
						continue
					}
					return fmt.Errorf("%s: global val '%s' initializer for type %s must be none", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
				}
				if isSupportedZeroedAggregateGlobalType(resolved, types, map[string]bool{}) {
					return fmt.Errorf("%s: global val '%s' initializer for type %s must be omitted", frontend.FormatPos(glob.Init.Pos()), glob.Name, resolved)
				}
				return fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
			}
			dataBlobs = append(dataBlobs, slotData...)
		}

		checked.GlobalsByModule[module] = globals
		checked.GlobalDataByModule[module] = dataBlobs
	}
	return nil
}
