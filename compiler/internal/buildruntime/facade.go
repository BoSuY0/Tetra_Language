package buildruntime

import (
	"tetra_language/compiler/internal/buildruntime/actors"
	"tetra_language/compiler/internal/buildruntime/linuxrt"
	"tetra_language/compiler/internal/buildruntime/selfhostrt"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func BuildActorDispatchFunc(
	entries []string,
	checked *semantics.CheckedProgram,
) (ir.IRFunc, error) {
	return actors.BuildActorDispatchFunc(entries, checked)
}

func BuildActorMainEntryIDFunc(mainName string) (ir.IRFunc, error) {
	return actors.BuildActorMainEntryIDFunc(mainName)
}

func ActorGlueNeeds(rt *tobj.Object) (dispatch bool, mainEntryID bool) {
	return actors.ActorGlueNeeds(rt)
}

func BuildActorGlueObject(
	rt *tobj.Object,
	target string,
	entries []string,
	checked *semantics.CheckedProgram,
	codegen func([]ir.IRFunc, [][]byte) (*tobj.Object, error),
) (*tobj.Object, bool, error) {
	return actors.BuildActorGlueObject(rt, target, entries, checked, codegen)
}

func CollectActorEntries(checked *semantics.CheckedProgram) (bool, []string, int, error) {
	return actors.CollectActorEntries(checked)
}

func CollectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return actors.CollectActorStateRuntimeUsage(checked)
}

func CollectActorStateRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return actors.CollectActorStateRuntimeUsagePosition(checked)
}

func CollectActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return actors.CollectActorRuntimeUsagePosition(checked)
}

func CollectActorSystemReceiveRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return actors.CollectActorSystemReceiveRuntimeUsagePosition(checked)
}

func CollectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return actors.CollectTaskRuntimeUsage(checked)
}

func CollectTaskRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return actors.CollectTaskRuntimeUsagePosition(checked)
}

func CollectTaskGroupRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return actors.CollectTaskGroupRuntimeUsage(checked)
}

func CollectTypedTaskRuntimeUsage(checked *semantics.CheckedProgram) (bool, int) {
	return actors.CollectTypedTaskRuntimeUsage(checked)
}

func BuildLinuxX86FilesystemRuntimeObject() *tobj.Object {
	return linuxrt.BuildLinuxX86FilesystemRuntimeObject()
}

func AppendLinuxX86FilesystemRuntimeObject(rt *tobj.Object) error {
	return linuxrt.AppendLinuxX86FilesystemRuntimeObject(rt)
}

func BuildLinuxX86BasicNetRuntimeObject() *tobj.Object {
	return linuxrt.BuildLinuxX86BasicNetRuntimeObject()
}

func AppendLinuxX86BasicNetRuntimeObject(rt *tobj.Object) error {
	return linuxrt.AppendLinuxX86BasicNetRuntimeObject(rt)
}

func BuildLinuxX32FilesystemRuntimeObject() *tobj.Object {
	return linuxrt.BuildLinuxX32FilesystemRuntimeObject()
}

func AppendLinuxX32FilesystemRuntimeObject(rt *tobj.Object) error {
	return linuxrt.AppendLinuxX32FilesystemRuntimeObject(rt)
}

func BuildLinuxX32BasicNetRuntimeObject() *tobj.Object {
	return linuxrt.BuildLinuxX32BasicNetRuntimeObject()
}

func AppendLinuxX32BasicNetRuntimeObject(rt *tobj.Object) error {
	return linuxrt.AppendLinuxX32BasicNetRuntimeObject(rt)
}

func BuildEmbeddedSelfHostRuntimeObject(
	target string,
	src []byte,
	filename string,
	codegen func([]ir.IRFunc, [][]byte) (*tobj.Object, error),
) (*tobj.Object, error) {
	return selfhostrt.BuildEmbeddedSelfHostRuntimeObject(target, src, filename, codegen)
}
