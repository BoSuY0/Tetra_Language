package semantics

import "tetra_language/compiler/internal/frontend"

const (
	surfaceSurfaceTypeName     = "lib.core.surface.Surface"
	surfaceFrameTypeName       = "lib.core.surface.Frame"
	surfaceEventTypeName       = "lib.core.surface.Event"
	surfaceDrawContextTypeName = "lib.core.draw.DrawContext"
)

func surfaceEphemeralValueType(typeName string, types map[string]*TypeInfo) (string, bool) {
	return surfaceEphemeralValueTypeVisiting(typeName, types, map[string]bool{})
}

func surfaceEphemeralValueTypeVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) (string, bool) {
	switch typeName {
	case surfaceFrameTypeName, surfaceEventTypeName, surfaceDrawContextTypeName:
		return typeName, true
	}
	if visiting[typeName] {
		return "", false
	}
	info, ok := types[typeName]
	if !ok {
		return "", false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if surfaceType, ok := surfaceEphemeralValueTypeVisiting(field.TypeName, types, visiting); ok {
				return surfaceType, true
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if surfaceType, ok := surfaceEphemeralValueTypeVisiting(payload, types, visiting); ok {
					return surfaceType, true
				}
			}
		}
	case TypeArray, TypeOptional, TypeSlice:
		return surfaceEphemeralValueTypeVisiting(info.ElemType, types, visiting)
	}
	return "", false
}

func surfaceActorTaskBoundaryValueType(typeName string, types map[string]*TypeInfo) (string, bool) {
	return surfaceActorTaskBoundaryValueTypeVisiting(typeName, types, map[string]bool{})
}

func surfaceAggregateFieldStorageAllowed(containerType string, fieldName string, fieldType string) bool {
	return containerType == surfaceDrawContextTypeName &&
		fieldName == "frame" &&
		fieldType == surfaceFrameTypeName
}

func surfaceActorTaskBoundaryValueTypeVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) (string, bool) {
	switch typeName {
	case surfaceSurfaceTypeName, surfaceFrameTypeName, surfaceEventTypeName, surfaceDrawContextTypeName:
		return typeName, true
	}
	if visiting[typeName] {
		return "", false
	}
	info, ok := types[typeName]
	if !ok {
		return "", false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if surfaceType, ok := surfaceActorTaskBoundaryValueTypeVisiting(field.TypeName, types, visiting); ok {
				return surfaceType, true
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if surfaceType, ok := surfaceActorTaskBoundaryValueTypeVisiting(payload, types, visiting); ok {
					return surfaceType, true
				}
			}
		}
	case TypeArray, TypeOptional, TypeSlice:
		return surfaceActorTaskBoundaryValueTypeVisiting(info.ElemType, types, visiting)
	}
	return "", false
}

func surfaceEphemeralReturnAllowed(analysis *functionAnalysisState, surfaceType string) bool {
	if analysis == nil {
		return false
	}
	switch analysis.currentFuncName {
	case "lib.core.surface.begin_frame":
		return surfaceType == surfaceFrameTypeName
	case "lib.core.surface.poll_event":
		return surfaceType == surfaceEventTypeName
	default:
		return false
	}
}

func surfaceFramePixelsEscapeExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, analysis *functionAnalysisState) bool {
	_, ok := surfaceFramePixelsSourceExpr(expr, locals, globals, types, analysis)
	return ok
}

func surfaceFramePixelsSourceExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, analysis *functionAnalysisState) (string, bool) {
	switch e := expr.(type) {
	case *frontend.FieldAccessExpr:
		if e.Field != "pixels" {
			return "", false
		}
		_, baseType, err := ResolveFieldAccessType(e.Base, locals, globals, types)
		if err != nil || baseType != surfaceFrameTypeName {
			return "", false
		}
		if path, ok := canonicalOwnershipAccessPath(e.Base); ok {
			return path, true
		}
		return "", true
	case *frontend.IdentExpr:
		if source, ok := analysis.localSurfaceFramePixelsSource(e.Name); ok {
			return source, true
		}
		if local, ok := locals[e.Name]; ok && local.SurfaceFramePixelsSource != "" {
			return local.SurfaceFramePixelsSource, true
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if source, ok := surfaceFramePixelsSourceExpr(field.Value, locals, globals, types, analysis); ok {
				return source, true
			}
		}
	}
	return "", false
}

func surfaceFrameOwnerSourceExpr(expr frontend.Expr, analysis *functionAnalysisState) (string, bool) {
	if call, ok := expr.(*frontend.CallExpr); ok && call.Name == "lib.core.surface.begin_frame" && len(call.Args) > 0 {
		return canonicalOwnershipAccessPath(call.Args[0])
	}
	if owner, ok := surfaceManualFrameOwnerSourceExpr(expr); ok {
		return owner, true
	}
	if path, ok := canonicalOwnershipAccessPath(expr); ok {
		return analysis.localSurfaceFrameOwner(path)
	}
	return "", false
}

func surfaceManualFrameOwnerSourceExpr(expr frontend.Expr) (string, bool) {
	var surfaceExpr frontend.Expr
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		if e.Type.Name != surfaceFrameTypeName {
			return "", false
		}
		for _, field := range e.Fields {
			if field.Name == "surface" {
				surfaceExpr = field.Value
				break
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType != surfaceFrameTypeName && e.Name != surfaceFrameTypeName {
			return "", false
		}
		for i, label := range e.ArgLabels {
			if label == "surface" && i < len(e.Args) {
				surfaceExpr = e.Args[i]
				break
			}
		}
		if surfaceExpr == nil && len(e.Args) > 0 {
			surfaceExpr = e.Args[0]
		}
	default:
		return "", false
	}
	if surfaceExpr == nil {
		return "", false
	}
	if owner, ok := canonicalOwnershipAccessPath(surfaceExpr); ok {
		return owner, true
	}
	return surfaceConstructedHandleOwnerPathExpr(surfaceExpr)
}

func surfaceHandleOwnerPathExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo) (string, bool) {
	return surfaceHandleOwnerPathExprWithAnalysis(expr, locals, globals, types, nil)
}

func surfaceHandleOwnerPathExprWithAnalysis(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, analysis *functionAnalysisState) (string, bool) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		return analysis.localSurfaceHandleOwner(id.Name)
	}
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field.Field != "handle" {
		return "", false
	}
	_, baseType, err := ResolveFieldAccessType(field.Base, locals, globals, types)
	if err != nil || baseType != surfaceSurfaceTypeName {
		return "", false
	}
	return canonicalOwnershipAccessPath(field.Base)
}

func surfaceHostABIHandleArgIndex(name string) (int, bool) {
	switch name {
	case "core.surface_close",
		"core.surface_poll_event_kind",
		"core.surface_poll_event_x",
		"core.surface_poll_event_y",
		"core.surface_poll_event_button",
		"core.surface_poll_event_into",
		"core.surface_poll_event_text_len",
		"core.surface_poll_event_text_into",
		"core.surface_clipboard_write_text",
		"core.surface_clipboard_read_text_into",
		"core.surface_poll_composition_into",
		"core.surface_begin_frame",
		"core.surface_present_rgba",
		"core.surface_request_redraw":
		return 0, true
	default:
		return 0, false
	}
}

func surfaceConstructedHandleOwnerPathExpr(expr frontend.Expr) (string, bool) {
	var handle frontend.Expr
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		if e.Type.Name != surfaceSurfaceTypeName {
			return "", false
		}
		for _, field := range e.Fields {
			if field.Name == "handle" {
				handle = field.Value
				break
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType != surfaceSurfaceTypeName && e.Name != surfaceSurfaceTypeName {
			return "", false
		}
		for i, label := range e.ArgLabels {
			if label == "handle" && i < len(e.Args) {
				handle = e.Args[i]
				break
			}
		}
		if handle == nil && len(e.Args) > 0 {
			handle = e.Args[0]
		}
	default:
		return "", false
	}
	if handle == nil {
		return "", false
	}
	field, ok := handle.(*frontend.FieldAccessExpr)
	if !ok || field.Field != "handle" {
		return "", false
	}
	return canonicalOwnershipAccessPath(field.Base)
}

func bindSurfaceFrameOwnerForLocal(name string, typeName string, expr frontend.Expr, analysis *functionAnalysisState) {
	if analysis == nil || name == "" {
		return
	}
	switch typeName {
	case surfaceFrameTypeName:
		if owner, ok := surfaceFrameOwnerSourceExpr(expr, analysis); ok {
			analysis.setLocalSurfaceFrameOwner(name, owner)
		} else {
			analysis.setLocalSurfaceFrameOwner(name, "")
		}
	case surfaceDrawContextTypeName:
		if owner, ok := surfaceDrawContextFrameOwnerSourceExpr(expr, analysis); ok {
			analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), owner)
		} else {
			analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), "")
		}
	default:
		analysis.setLocalSurfaceFrameOwner(name, "")
		analysis.setLocalSurfaceFrameOwner(resourceFieldPath(name, "frame"), "")
	}
}

func surfaceDrawContextFrameOwnerSourceExpr(expr frontend.Expr, analysis *functionAnalysisState) (string, bool) {
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if field.Name == "frame" {
				return surfaceFrameOwnerSourceExpr(field.Value, analysis)
			}
		}
	case *frontend.CallExpr:
		if e.Name == surfaceDrawContextTypeName && len(e.Args) > 0 {
			return surfaceFrameOwnerSourceExpr(e.Args[0], analysis)
		}
	default:
		if path, ok := canonicalOwnershipAccessPath(expr); ok {
			return analysis.localSurfaceFrameOwner(resourceFieldPath(path, "frame"))
		}
	}
	return "", false
}

func surfacePresentedFrameArg(expr frontend.Expr) (string, bool) {
	return canonicalOwnershipAccessPath(expr)
}

func checkSurfacePresentFrameOwner(expr frontend.Expr, analysis *functionAnalysisState, state *regionState, pos frontend.Position) error {
	frameName, ok := surfacePresentedFrameArg(expr)
	if !ok {
		return nil
	}
	return checkSurfacePresentFrameOwnerPath(frameName, analysis, state, pos)
}

func checkSurfacePresentFrameOwnerPath(frameName string, analysis *functionAnalysisState, state *regionState, pos frontend.Position) error {
	if frameName == "" {
		return nil
	}
	owner, ok := analysis.localSurfaceFrameOwner(frameName)
	if !ok || owner == "" {
		return nil
	}
	return state.checkNotConsumed(owner, pos)
}

func isSurfacePresentCallName(name string) bool {
	return name == "lib.core.surface.present"
}
