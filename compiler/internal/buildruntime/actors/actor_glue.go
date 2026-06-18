package actors

import (
	"fmt"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/semantics"
)

func ActorGlueNeeds(rt *tobj.Object) (dispatch bool, mainEntryID bool) {
	dispatch = true
	mainEntryID = true
	if rt == nil {
		return dispatch, mainEntryID
	}
	for _, sym := range rt.Symbols {
		if sym.Name == "__tetra_actor_dispatch" {
			dispatch = false
		}
		if sym.Name == "__tetra_actor_main_entry_id" {
			mainEntryID = false
		}
	}
	return dispatch, mainEntryID
}

func BuildActorGlueObject(
	rt *tobj.Object,
	target string,
	entries []string,
	checked *semantics.CheckedProgram,
	codegen func([]ir.IRFunc, [][]byte) (*tobj.Object, error),
) (*tobj.Object, bool, error) {
	needsDispatchGlue, needsMainEntryIDGlue := ActorGlueNeeds(rt)
	if !needsDispatchGlue && !needsMainEntryIDGlue {
		return nil, false, nil
	}

	var glueFuncs []ir.IRFunc
	if needsDispatchGlue {
		dispatchFn, err := BuildActorDispatchFunc(entries, checked)
		if err != nil {
			return nil, false, err
		}
		glueFuncs = append(glueFuncs, dispatchFn)
	}
	if needsMainEntryIDGlue {
		if len(entries) == 0 {
			return nil, false, fmt.Errorf("missing actor entries")
		}
		mainIDFn, err := BuildActorMainEntryIDFunc(entries[0])
		if err != nil {
			return nil, false, err
		}
		glueFuncs = append(glueFuncs, mainIDFn)
	}
	for _, fn := range glueFuncs {
		if err := lower.VerifyFunc(fn); err != nil {
			return nil, false, fmt.Errorf("generated actor glue verifier: %w", err)
		}
	}
	glueObj, err := codegen(glueFuncs, nil)
	if err != nil {
		return nil, false, err
	}
	glueObj.Target = target
	glueObj.Module = "__actorsglue"
	return glueObj, true, nil
}
