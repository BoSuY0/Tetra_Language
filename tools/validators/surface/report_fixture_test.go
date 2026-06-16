package surface

func componentMap(id string, typ string, parent string, bounds RectReport, state map[string]string) map[string]any {
	value := map[string]any{
		"id":        id,
		"type":      typ,
		"bounds":    rectMap(bounds),
		"abilities": []any{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		"state":     stringMapAny(state),
	}
	if parent != "" {
		value["parent"] = parent
	}
	return value
}
func treeNodeMap(id int, name string, kind string, parentID int, childIndex int, firstChild int, childCount int, focusable bool, bounds RectReport) map[string]any {
	return map[string]any{"id": id, "name": name, "kind": kind, "parent_id": parentID, "child_index": childIndex, "first_child": firstChild, "child_count": childCount, "focusable": focusable, "bounds": rectMap(bounds)}
}
func rectMap(rect RectReport) map[string]any {
	return map[string]any{"x": rect.X, "y": rect.Y, "w": rect.W, "h": rect.H}
}
func toolkitWidgetMap(name string, kind string, nodeID int, role string, reusable bool) map[string]any {
	value := map[string]any{"name": name, "kind": kind, "node_id": nodeID, "reusable": reusable, "ordinary_tetra_struct": true}
	if role != "" {
		if kind == "Button" {
			value["action"] = role
		} else {
			value["role"] = role
		}
	}
	if kind == "TextBox" {
		value["editable"] = true
	}
	return value
}
func eventMap(order int, kind string, target string, path []any, x int, y int, key int, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": kind, "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": x, "y": y, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{5, x, y, 1, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}
func keyEventMap(order int, target string, path []any, key int, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "key_down", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{6, 0, 0, 0, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}
func textEventMap(order int, target string, path []any, textLen int, textHex string, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "text_input", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "text_len": textLen, "text_bytes_hex": textHex,
		"buffer_slots": []any{8, 0, 0, 0, 0, width, height, order - 1, textLen},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}
func resizeEventMap(order int, target string, path []any, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "resize", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{2, 0, 0, 0, 0, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}
func stringMapAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
func treeComponent(id string, typ string, parent string, bounds RectReport, state map[string]string) ComponentReport {
	return ComponentReport{
		ID:        id,
		Type:      typ,
		Parent:    parent,
		Bounds:    bounds,
		Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		State:     state,
	}
}
func intPtrForTest(v int) *int {
	return &v
}
