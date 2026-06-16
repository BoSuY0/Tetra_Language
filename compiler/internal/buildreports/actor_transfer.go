package buildreports

import (
	"sort"
	"strings"

	"tetra_language/compiler/internal/actorsafety"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func BuildActorTransferReport(checked *semantics.CheckedProgram, target string) ActorTransferReport {
	report := ActorTransferReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 1, Kind: "actor_transfer", Target: target},
	}
	if checked == nil {
		return report
	}
	mailboxes := map[string]ActorMailboxRow{}
	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		walkActorTransferStmts(fn.Decl.Body, func(call *frontend.CallExpr) {
			if msgType := actorMailboxMessageTypeFromCall(call, checked.Types); msgType != "" {
				if row, ok := actorMailboxRowForMessage(msgType, checked.Types, target); ok {
					mailboxes[row.Name] = row
				}
			}
			rows := actorTransferRowsForSend(fn.Name, call, checked.Types)
			for _, row := range rows {
				switch row.TransferMode {
				case "copy":
					report.Totals.Copy++
				case "move":
					report.Totals.Move++
				case "zero_copy_move":
					report.Totals.ZeroCopyMove++
				}
				report.Totals.BytesCopied += row.BytesCopied
				report.Sends = append(report.Sends, row)
			}
		})
	}
	if len(mailboxes) > 0 {
		names := make([]string, 0, len(mailboxes))
		for name := range mailboxes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			report.Mailboxes = append(report.Mailboxes, mailboxes[name])
		}
	}
	return report
}

func actorMailboxMessageTypeFromCall(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) string {
	if call == nil {
		return ""
	}
	switch call.Name {
	case "core.send_typed":
		if len(call.Args) < 2 {
			return ""
		}
		msgCall, ok := call.Args[1].(*frontend.CallExpr)
		if !ok {
			return ""
		}
		msgType, _, ok := reportEnumCaseConstructor(msgCall, types)
		if ok {
			return msgType
		}
	case "core.recv_typed":
		if len(call.TypeArgs) == 1 {
			return call.TypeArgs[0].Name
		}
	}
	return ""
}

func actorMailboxRowForMessage(typeName string, types map[string]*semantics.TypeInfo, target string) (ActorMailboxRow, bool) {
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return ActorMailboxRow{}, false
	}
	row := ActorMailboxRow{
		Name:              "typed:" + typeName,
		MessageSchema:     typeName,
		Capacity:          actorMailboxCapacityForTarget(target),
		CapacityUnit:      "messages",
		Backpressure:      "blocking_recv_yield",
		OverflowPolicy:    "unchecked_fixed_pool_overflow",
		MaxPayloadSlots:   8,
		PayloadSlots:      info.SlotCount - 1,
		SlotWidthBytes:    actorMailboxSlotWidthBytes(target),
		RuntimePath:       "actor_mailbox_typed_slots",
		OwnershipMetadata: true,
	}
	if err := actorsafety.VerifyMailbox(actorsafety.Mailbox{
		Name:         row.Name,
		Message:      row.MessageSchema,
		Capacity:     row.Capacity,
		Backpressure: row.Backpressure,
	}); err != nil {
		return ActorMailboxRow{}, false
	}
	return row, true
}

func actorMailboxCapacityForTarget(target string) int {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return 64 * 1024 / 88
	default:
		return 64 * 1024 / 88
	}
}

func actorMailboxSlotWidthBytes(target string) int {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return 8
	default:
		return 8
	}
}

func actorTransferRowsForSend(function string, call *frontend.CallExpr, types map[string]*semantics.TypeInfo) []ActorTransferRow {
	if call == nil || call.Name != "core.send_typed" || len(call.Args) < 2 {
		return nil
	}
	msgCall, ok := call.Args[1].(*frontend.CallExpr)
	if !ok {
		return nil
	}
	msgType, caseInfo, ok := reportEnumCaseConstructor(msgCall, types)
	if !ok {
		return nil
	}
	owners := make([]string, 0, len(caseInfo.PayloadTypes))
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			break
		}
		if reportTypeKind(payloadType, types) == semantics.TypeIsland {
			if owner := reportExprPath(msgCall.Args[i]); owner != "" {
				owners = append(owners, owner)
			}
		}
	}
	rows := []ActorTransferRow{}
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			continue
		}
		if row, ok := actorTransferRowForPayload(function, call, msgType, caseInfo, i, payloadType, msgCall.Args[i], owners, types); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func actorTransferRowForPayload(
	function string,
	call *frontend.CallExpr,
	msgType string,
	caseInfo semantics.EnumCaseInfo,
	index int,
	payloadType string,
	expr frontend.Expr,
	owners []string,
	types map[string]*semantics.TypeInfo,
) (ActorTransferRow, bool) {
	base := ActorTransferRow{
		Function:                   function,
		Site:                       reportPos(call.At),
		MessageType:                msgType,
		Case:                       caseInfo.Name,
		PayloadIndex:               index,
		PayloadType:                payloadType,
		ClaimLevel:                 "validated",
		BoundaryScope:              "local_typed_mailbox",
		ProductionRuntimeValidated: false,
	}
	slotBytes := reportPayloadSlotCount(caseInfo, index, payloadType, types) * actorMailboxSlotWidthBytes("")
	switch reportTypeKind(payloadType, types) {
	case semantics.TypeI32, semantics.TypeU8, semantics.TypeBool:
		base.Ownership = "copy"
		base.TransferMode = "copy"
		base.RuntimePath = "actor_mailbox_value_slot"
		base.BytesCopied = slotBytes
		base.ZeroCopy = false
		base.Reason = "small scalar payload crosses typed actor mailbox by copy"
		return base, true
	case semantics.TypeIsland:
		base.Ownership = "owned_region"
		base.Owner = reportExprPath(expr)
		base.TransferMode = "move"
		base.RuntimePath = "actor_mailbox_resource_slot"
		base.BytesCopied = 0
		base.ZeroCopy = true
		base.ClaimLevel = "evidence_only"
		base.BoundaryScope = "local_typed_mailbox_owned_region_move"
		base.Reason = "island payload moves ownership across typed actor mailbox"
		return base, true
	case semantics.TypeStr, semantics.TypeSlice:
		if reportExprIsExplicitCopy(expr) {
			base.Ownership = "owned_copy"
			base.TransferMode = "copy"
			base.RuntimePath = "actor_mailbox_copy_region_slot"
			base.BytesCopied = slotBytes
			base.ZeroCopy = false
			base.Reason = "borrowed view crosses actor boundary through explicit copy"
			return base, true
		}
		if reportTypeKind(payloadType, types) == semantics.TypeSlice {
			owner := reportOwnedRegionSliceOwner(expr, owners)
			if owner == "" {
				return ActorTransferRow{}, false
			}
			base.Ownership = "owned_region_slice"
			base.Owner = owner
			base.TransferMode = "zero_copy_move"
			base.RuntimePath = "actor_mailbox_zero_copy_region_slot"
			base.BytesCopied = 0
			base.ZeroCopy = true
			base.ClaimLevel = "evidence_only"
			base.BoundaryScope = "local_typed_mailbox_owned_region_slice_move"
			base.Reason = "owned region-backed slice moves with its island owner in the same typed actor payload"
			return base, true
		}
	case semantics.TypeStruct, semantics.TypeEnum:
		base.Ownership = "copy"
		base.TransferMode = "copy"
		base.RuntimePath = "actor_mailbox_aggregate_value_slots"
		base.BytesCopied = slotBytes
		base.ZeroCopy = false
		base.Reason = "value-only aggregate payload crosses typed actor mailbox by slot copy"
		return base, true
	}
	return ActorTransferRow{}, false
}

func reportPayloadSlotCount(caseInfo semantics.EnumCaseInfo, index int, payloadType string, types map[string]*semantics.TypeInfo) int {
	if index >= 0 && index < len(caseInfo.PayloadSlots) && caseInfo.PayloadSlots[index] > 0 {
		return caseInfo.PayloadSlots[index]
	}
	if info, ok := types[payloadType]; ok && info.SlotCount > 0 {
		return info.SlotCount
	}
	return 1
}

func reportEnumCaseConstructor(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) (string, semantics.EnumCaseInfo, bool) {
	if call == nil {
		return "", semantics.EnumCaseInfo{}, false
	}
	if call.ResolvedType != "" {
		if info, ok := types[call.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
			caseName := reportCallCaseName(call.Name)
			if caseInfo, ok := info.CaseMap[caseName]; ok {
				return call.ResolvedType, caseInfo, true
			}
		}
	}
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", semantics.EnumCaseInfo{}, false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	caseName := parts[len(parts)-1]
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		return "", semantics.EnumCaseInfo{}, false
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", semantics.EnumCaseInfo{}, false
	}
	return typeName, caseInfo, true
}

func reportCallCaseName(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

func reportTypeKind(typeName string, types map[string]*semantics.TypeInfo) semantics.TypeKind {
	if info, ok := types[typeName]; ok {
		return info.Kind
	}
	return 0
}

func reportOwnedRegionSliceOwner(expr frontend.Expr, owners []string) string {
	if len(owners) == 0 || expr == nil || reportExprIsExplicitCopy(expr) {
		return ""
	}
	if call, ok := expr.(*frontend.CallExpr); ok && len(call.Args) > 0 && reportIsIslandMakeCall(call.Name) {
		owner := reportExprPath(call.Args[0])
		if reportStringIn(owner, owners) {
			return owner
		}
		return ""
	}
	return owners[0]
}

func reportIsIslandMakeCall(name string) bool {
	switch name {
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool",
		"island_make_u8", "island_make_u16", "island_make_i32", "island_make_bool":
		return true
	default:
		return false
	}
}

func reportExprIsExplicitCopy(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if name == "core.string_copy" || name == "string_copy" {
		return true
	}
	return (strings.HasPrefix(name, "core.slice_copy_") || strings.HasPrefix(name, "slice_copy_")) &&
		!strings.HasPrefix(name, "core.slice_copy_into_") &&
		!strings.HasPrefix(name, "slice_copy_into_")
}

func reportExprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := reportExprPath(e.Base)
		if base == "" || e.Field == "" {
			return base
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func reportStringIn(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func walkActorTransferStmts(stmts []frontend.Stmt, visit func(*frontend.CallExpr)) {
	for _, stmt := range stmts {
		walkActorTransferStmt(stmt, visit)
	}
}

func walkActorTransferStmt(stmt frontend.Stmt, visit func(*frontend.CallExpr)) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.ReturnStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.ThrowStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.DeferStmt:
		walkActorTransferStmts(s.Body, visit)
	case *frontend.LetStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.AssignStmt:
		walkActorTransferExpr(s.Target, visit)
		walkActorTransferExpr(s.Value, visit)
		walkActorTransferExpr(s.CompoundValue, visit)
	case *frontend.IfStmt:
		walkActorTransferExpr(s.Cond, visit)
		walkActorTransferStmts(s.Then, visit)
		walkActorTransferStmts(s.Else, visit)
	case *frontend.IfLetStmt:
		walkActorTransferExpr(s.Pattern, visit)
		walkActorTransferExpr(s.Value, visit)
		walkActorTransferStmts(s.Then, visit)
		walkActorTransferStmts(s.Else, visit)
	case *frontend.WhileStmt:
		walkActorTransferExpr(s.Cond, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.ForRangeStmt:
		walkActorTransferExpr(s.Start, visit)
		walkActorTransferExpr(s.End, visit)
		walkActorTransferExpr(s.Iterable, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.MatchStmt:
		walkActorTransferExpr(s.Value, visit)
		for _, c := range s.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferStmts(c.Body, visit)
		}
	case *frontend.FreeStmt:
		walkActorTransferExpr(s.Value, visit)
	case *frontend.UnsafeStmt:
		walkActorTransferStmts(s.Body, visit)
	case *frontend.IslandStmt:
		walkActorTransferExpr(s.Size, visit)
		walkActorTransferStmts(s.Body, visit)
	case *frontend.ExprStmt:
		walkActorTransferExpr(s.Expr, visit)
	case *frontend.ExpectStmt:
		walkActorTransferExpr(s.Cond, visit)
	}
}

func walkActorTransferExpr(expr frontend.Expr, visit func(*frontend.CallExpr)) {
	switch e := expr.(type) {
	case nil:
		return
	case *frontend.BinaryExpr:
		walkActorTransferExpr(e.Left, visit)
		walkActorTransferExpr(e.Right, visit)
	case *frontend.UnaryExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.TryExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.AwaitExpr:
		walkActorTransferExpr(e.X, visit)
	case *frontend.CallExpr:
		visit(e)
		for _, arg := range e.Args {
			walkActorTransferExpr(arg, visit)
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			walkActorTransferStmts(e.Decl.Body, visit)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			walkActorTransferExpr(field.Value, visit)
		}
	case *frontend.FieldAccessExpr:
		walkActorTransferExpr(e.Base, visit)
	case *frontend.IndexExpr:
		walkActorTransferExpr(e.Base, visit)
		walkActorTransferExpr(e.Index, visit)
	case *frontend.MatchExpr:
		walkActorTransferExpr(e.Value, visit)
		for _, c := range e.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferExpr(c.Value, visit)
		}
	case *frontend.CatchExpr:
		walkActorTransferExpr(e.Call, visit)
		for _, c := range e.Cases {
			walkActorTransferExpr(c.Pattern, visit)
			walkActorTransferExpr(c.Guard, visit)
			walkActorTransferExpr(c.Value, visit)
		}
	}
}
