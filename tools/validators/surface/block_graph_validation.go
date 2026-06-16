package surface

import (
	"fmt"
	"sort"
	"strings"
)

type BlockGraphReport struct {
	Schema             string                       `json:"schema"`
	APILevel           string                       `json:"api_level"`
	Source             string                       `json:"source"`
	ManualBookkeeping  bool                         `json:"manual_bookkeeping"`
	Builder            BlockGraphBuilderReport      `json:"builder"`
	Invariants         BlockGraphInvariantReport    `json:"invariants"`
	RootID             int                          `json:"root_id"`
	NodeCount          int                          `json:"node_count"`
	Nodes              []BlockGraphNodeReport       `json:"nodes"`
	ChildOrders        []BlockGraphChildOrderReport `json:"child_orders"`
	LayoutOrder        []int                        `json:"layout_order"`
	DrawOrder          []int                        `json:"draw_order"`
	FocusOrder         []int                        `json:"focus_order"`
	AccessibilityOrder []int                        `json:"accessibility_order"`
	HitTests           []BlockGraphPathReport       `json:"hit_tests"`
	DispatchPaths      []BlockGraphPathReport       `json:"dispatch_paths"`
}

type BlockGraphBuilderReport struct {
	RootCreatedBy     string `json:"root_created_by"`
	ChildrenCreatedBy string `json:"children_created_by"`
	NodeCount         int    `json:"node_count"`
	Capacity          int    `json:"capacity"`
	OverflowChecked   bool   `json:"overflow_checked"`
}

type BlockGraphInvariantReport struct {
	TreeValidateRan         bool `json:"tree_validate_ran"`
	TreeValidateStatus      int  `json:"tree_validate_status"`
	DuplicateIDRejected     bool `json:"duplicate_id_rejected"`
	MissingParentRejected   bool `json:"missing_parent_rejected"`
	CycleRejected           bool `json:"cycle_rejected"`
	ParentChildLinksChecked bool `json:"parent_child_links_checked"`
	ChildOrderChecked       bool `json:"child_order_checked"`
	FocusOrderChecked       bool `json:"focus_order_checked"`
	HitTestPathChecked      bool `json:"hit_test_path_checked"`
	AccessibilityChecked    bool `json:"accessibility_order_checked"`
}

type BlockGraphNodeReport struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	ParentID          int        `json:"parent_id"`
	ChildIndex        int        `json:"child_index"`
	FirstChild        int        `json:"first_child"`
	ChildCount        int        `json:"child_count"`
	Focusable         bool       `json:"focusable"`
	AccessibilityRole string     `json:"accessibility_role"`
	Bounds            RectReport `json:"bounds"`
}

type BlockGraphChildOrderReport struct {
	ParentID int   `json:"parent_id"`
	Children []int `json:"children"`
}

type BlockGraphPathReport struct {
	Helper   string `json:"helper"`
	Event    string `json:"event,omitempty"`
	TargetID int    `json:"target_id"`
	X        int    `json:"x,omitempty"`
	Y        int    `json:"y,omitempty"`
	Path     []int  `json:"path"`
}

func validateBlockGraphEvidence(report Report) []string {
	if report.BlockGraph == nil {
		return nil
	}

	graph := report.BlockGraph
	var issues []string
	if graph.Schema != "tetra.surface.block-graph.v1" {
		issues = append(issues, fmt.Sprintf("block_graph schema is %q, want tetra.surface.block-graph.v1", graph.Schema))
	}
	if graph.APILevel != "block-tree-builder-v1" {
		issues = append(issues, fmt.Sprintf("block_graph api_level is %q, want block-tree-builder-v1", graph.APILevel))
	}
	if normalizeEvidencePath(graph.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_graph source %q must match report source %q", graph.Source, report.Source))
	}
	if graph.ManualBookkeeping {
		issues = append(issues, "block_graph manual_bookkeeping must be false")
	}
	if graph.Builder.RootCreatedBy != "tree_add_root" {
		issues = append(issues, fmt.Sprintf("block_graph builder root_created_by is %q, want tree_add_root", graph.Builder.RootCreatedBy))
	}
	if graph.Builder.ChildrenCreatedBy != "tree_add_child" {
		issues = append(issues, fmt.Sprintf("block_graph builder children_created_by is %q, want tree_add_child", graph.Builder.ChildrenCreatedBy))
	}
	if graph.Builder.NodeCount != graph.NodeCount {
		issues = append(issues, fmt.Sprintf("block_graph builder node_count = %d, want block_graph node_count %d", graph.Builder.NodeCount, graph.NodeCount))
	}
	if graph.Builder.Capacity < graph.NodeCount {
		issues = append(issues, fmt.Sprintf("block_graph builder capacity = %d, want at least node_count %d", graph.Builder.Capacity, graph.NodeCount))
	}
	if !graph.Builder.OverflowChecked {
		issues = append(issues, "block_graph builder must prove overflow_checked")
	}
	if !graph.Invariants.TreeValidateRan {
		issues = append(issues, "block_graph invariants require tree_validate_ran")
	}
	if graph.Invariants.TreeValidateStatus != 0 {
		issues = append(issues, fmt.Sprintf("block_graph tree_validate_status = %d, want 0", graph.Invariants.TreeValidateStatus))
	}
	if !graph.Invariants.DuplicateIDRejected {
		issues = append(issues, "block_graph invariants require duplicate_id_rejected")
	}
	if !graph.Invariants.MissingParentRejected {
		issues = append(issues, "block_graph invariants require missing_parent_rejected")
	}
	if !graph.Invariants.CycleRejected {
		issues = append(issues, "block_graph invariants require cycle_rejected")
	}
	if !graph.Invariants.ParentChildLinksChecked || !graph.Invariants.ChildOrderChecked || !graph.Invariants.FocusOrderChecked ||
		!graph.Invariants.HitTestPathChecked || !graph.Invariants.AccessibilityChecked {
		issues = append(issues, "block_graph invariants must check parent/child links, child order, focus order, hit-test path, and accessibility order")
	}
	if graph.NodeCount != len(graph.Nodes) {
		issues = append(issues, fmt.Sprintf("block_graph node_count = %d, want len(nodes) %d", graph.NodeCount, len(graph.Nodes)))
	}
	if graph.NodeCount < 5 {
		issues = append(issues, fmt.Sprintf("block_graph node_count = %d, want at least 5", graph.NodeCount))
	}

	nodes := map[int]BlockGraphNodeReport{}
	childrenByParent := map[int][]BlockGraphNodeReport{}
	for _, node := range graph.Nodes {
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("block_graph duplicate node id %d", node.ID))
		}
		nodes[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_graph node %d name is required", node.ID))
		}
		if node.Bounds.W < 0 || node.Bounds.H < 0 {
			issues = append(issues, fmt.Sprintf("block_graph node %d bounds must be non-negative", node.ID))
		}
		if node.ChildCount < 0 {
			issues = append(issues, fmt.Sprintf("block_graph node %d child_count must be non-negative", node.ID))
		}
		if node.ChildCount == 0 && node.FirstChild != -1 {
			issues = append(issues, fmt.Sprintf("block_graph leaf node %d first_child = %d, want -1", node.ID, node.FirstChild))
		}
		if node.ParentID >= 0 {
			childrenByParent[node.ParentID] = append(childrenByParent[node.ParentID], node)
		}
	}

	root, ok := nodes[graph.RootID]
	if !ok {
		issues = append(issues, fmt.Sprintf("block_graph root_id %d is not in nodes", graph.RootID))
	} else if root.ParentID != -1 {
		issues = append(issues, fmt.Sprintf("block_graph root %d parent_id = %d, want -1", root.ID, root.ParentID))
	}

	for _, node := range graph.Nodes {
		if node.ParentID < 0 {
			continue
		}
		parent, ok := nodes[node.ParentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_graph node %d parent_id %d is unknown", node.ID, node.ParentID))
			continue
		}
		if !rectContainsRect(parent.Bounds, node.Bounds) {
			issues = append(issues, fmt.Sprintf("block_graph node %d bounds must be inside parent %d bounds", node.ID, parent.ID))
		}
		if _, ok := blockGraphPathToRoot(node.ID, nodes); !ok {
			issues = append(issues, fmt.Sprintf("block_graph node %d has missing parent or cycle in root path", node.ID))
		}
	}

	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		sort.Slice(children, func(i, j int) bool {
			return children[i].ChildIndex < children[j].ChildIndex
		})
		if parent.ChildCount != len(children) {
			issues = append(issues, fmt.Sprintf("block_graph node %d child_count = %d, want %d", parentID, parent.ChildCount, len(children)))
		}
		expectedChildren := make([]int, 0, len(children))
		seenIndex := map[int]int{}
		for _, child := range children {
			if child.ChildIndex < 0 || child.ChildIndex >= len(children) {
				issues = append(issues, fmt.Sprintf("block_graph child node %d child_index = %d, want 0..%d", child.ID, child.ChildIndex, len(children)-1))
			}
			if prev, exists := seenIndex[child.ChildIndex]; exists {
				issues = append(issues, fmt.Sprintf("block_graph sibling child_index %d is used by nodes %d and %d", child.ChildIndex, prev, child.ID))
			}
			seenIndex[child.ChildIndex] = child.ID
			expectedChildren = append(expectedChildren, child.ID)
		}
		if len(expectedChildren) > 0 && parent.FirstChild != expectedChildren[0] {
			issues = append(issues, fmt.Sprintf("block_graph node %d first_child = %d, want %d", parentID, parent.FirstChild, expectedChildren[0]))
		}
		if !hasBlockGraphChildOrder(graph.ChildOrders, parentID, expectedChildren) {
			issues = append(issues, fmt.Sprintf("block_graph child_orders require parent %d children %v", parentID, expectedChildren))
		}
	}

	if !blockGraphOrderCoversNodes(graph.LayoutOrder, nodes) {
		issues = append(issues, "block_graph layout_order must include every node exactly once")
	}
	if !blockGraphOrderCoversNodes(graph.DrawOrder, nodes) {
		issues = append(issues, "block_graph draw_order must include every node exactly once")
	}

	expectedFocus := blockGraphFocusOrder(graph.Nodes)
	if !intSlicesEqual(graph.FocusOrder, expectedFocus) {
		issues = append(issues, fmt.Sprintf("block_graph focus_order = %v, want focusable node order %v", graph.FocusOrder, expectedFocus))
	}
	expectedAccessibility := blockGraphAccessibilityOrder(graph.Nodes)
	if !intSlicesEqual(graph.AccessibilityOrder, expectedAccessibility) {
		issues = append(issues, fmt.Sprintf("block_graph accessibility_order = %v, want accessible node order %v", graph.AccessibilityOrder, expectedAccessibility))
	}
	issues = append(issues, validateBlockGraphPaths("hit_tests", "tree_hit_test_path", graph.HitTests, nodes)...)
	issues = append(issues, validateBlockGraphPaths("dispatch_paths", "tree_build_dispatch_path", graph.DispatchPaths, nodes)...)

	for _, required := range []string{
		"block graph duplicate id rejected",
		"block graph missing parent rejected",
		"block graph cycle rejected",
		"block graph child order",
		"block graph focus order",
		"block graph hit-test path",
		"block graph accessibility order",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_graph report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockGraphChildOrder(orders []BlockGraphChildOrderReport, parentID int, expected []int) bool {
	for _, order := range orders {
		if order.ParentID == parentID && intSlicesEqual(order.Children, expected) {
			return true
		}
	}
	return false
}

func blockGraphOrderCoversNodes(order []int, nodes map[int]BlockGraphNodeReport) bool {
	if len(order) != len(nodes) {
		return false
	}
	seen := map[int]bool{}
	for _, id := range order {
		if _, ok := nodes[id]; !ok || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func blockGraphFocusOrder(nodes []BlockGraphNodeReport) []int {
	var order []int
	for _, node := range nodes {
		if node.Focusable {
			order = append(order, node.ID)
		}
	}
	return order
}

func blockGraphAccessibilityOrder(nodes []BlockGraphNodeReport) []int {
	var order []int
	for _, node := range nodes {
		role := strings.TrimSpace(strings.ToLower(node.AccessibilityRole))
		if role != "" && role != "none" {
			order = append(order, node.ID)
		}
	}
	return order
}

func blockGraphPathToRoot(id int, nodes map[int]BlockGraphNodeReport) ([]int, bool) {
	var reversed []int
	seen := map[int]bool{}
	for {
		node, ok := nodes[id]
		if !ok || seen[id] {
			return nil, false
		}
		seen[id] = true
		reversed = append(reversed, id)
		if node.ParentID < 0 {
			break
		}
		id = node.ParentID
	}
	path := make([]int, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path, true
}

func validateBlockGraphPaths(field string, helper string, paths []BlockGraphPathReport, nodes map[int]BlockGraphNodeReport) []string {
	if len(paths) == 0 {
		return []string{fmt.Sprintf("block_graph %s evidence is required", field)}
	}
	var issues []string
	for _, path := range paths {
		if path.Helper != helper {
			issues = append(issues, fmt.Sprintf("block_graph %s helper is %q, want %s", field, path.Helper, helper))
		}
		wantPath, ok := blockGraphPathToRoot(path.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("block_graph %s target_id %d is not reachable from root", field, path.TargetID))
			continue
		}
		if !intSlicesEqual(path.Path, wantPath) {
			issues = append(issues, fmt.Sprintf("block_graph %s target_id %d path = %v, want %v", field, path.TargetID, path.Path, wantPath))
		}
	}
	return issues
}
