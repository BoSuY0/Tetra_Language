package buildruntime

import (
	"fmt"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/runtimeabi"
)

func RequiredActorRuntimeSymbols() []string {
	return runtimeabi.RequiredActorSymbols()
}

func RequiredActorLifecycleRuntimeSymbols() []string {
	return runtimeabi.RequiredActorLifecycleSymbols()
}

func RequiredActorSystemReceiveRuntimeSymbols() []string {
	return runtimeabi.RequiredActorSystemReceiveSymbols()
}

func RequiredActorTelemetryRuntimeSymbols() []string {
	return runtimeabi.RequiredActorTelemetrySymbols()
}

func RequiredActorStateRuntimeSymbols() []string {
	return runtimeabi.RequiredActorStateSymbols()
}

func RequiredDistributedActorRuntimeSymbols() []string {
	return runtimeabi.RequiredDistributedActorSymbols()
}

func RequiredTaskRuntimeSymbols() []string {
	return runtimeabi.RequiredTaskSymbols()
}

func RequiredTaskGroupRuntimeSymbols() []string {
	return runtimeabi.RequiredTaskGroupSymbols()
}

func RequiredTypedTaskRuntimeSymbols(maxSlots int) []string {
	return runtimeabi.RequiredTypedTaskSymbols(maxSlots)
}

func RequiredTimeRuntimeSymbols() []string {
	return runtimeabi.RequiredTimeSymbols()
}

func RequiredFilesystemRuntimeSymbols() []string {
	return runtimeabi.RequiredFilesystemSymbols()
}

func RequiredNetRuntimeSymbols() []string {
	return runtimeabi.RequiredNetSymbols()
}

func RequiredSurfaceRuntimeSymbols() []string {
	return runtimeabi.RequiredSurfaceSymbols()
}

type RuntimeObjectSlotSignature struct {
	ParamSlots  int
	ReturnSlots int
}

func RuntimeObjectSignature(name string) (RuntimeObjectSlotSignature, bool) {
	sig, ok := runtimeabi.SignatureForSymbol(name)
	if !ok {
		return RuntimeObjectSlotSignature{}, false
	}
	return RuntimeObjectSlotSignature{
		ParamSlots:  sig.ParamSlots,
		ReturnSlots: sig.ReturnSlots,
	}, true
}

func AnnotateRuntimeObjectSignatures(rt *tobj.Object) {
	if rt == nil {
		return
	}
	for i := range rt.Symbols {
		if rt.Symbols[i].HasSignature {
			continue
		}
		sig, ok := RuntimeObjectSignature(rt.Symbols[i].Name)
		if !ok {
			continue
		}
		rt.Symbols[i].HasSignature = true
		rt.Symbols[i].ParamSlots = sig.ParamSlots
		rt.Symbols[i].ReturnSlots = sig.ReturnSlots
	}
}

func ValidateRuntimeObjectSymbols(rt *tobj.Object, missingObject string, required []string) error {
	if rt == nil {
		return fmt.Errorf("%s", missingObject)
	}
	symbols := make(map[string]tobj.Symbol, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = sym
	}
	for _, name := range required {
		sym, ok := symbols[name]
		if !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
		expected, ok := RuntimeObjectSignature(name)
		if !ok || !sym.HasSignature {
			continue
		}
		if sym.ParamSlots != expected.ParamSlots || sym.ReturnSlots != expected.ReturnSlots {
			return fmt.Errorf(
				"runtime object symbol '%s' signature mismatch: params=%d want=%d returns=%d want=%d",
				name,
				sym.ParamSlots,
				expected.ParamSlots,
				sym.ReturnSlots,
				expected.ReturnSlots,
			)
		}
	}
	return nil
}

func ValidateActorRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing actors runtime object",
		RequiredActorRuntimeSymbols(),
	)
}

func ValidateActorLifecycleRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing actor lifecycle runtime object",
		RequiredActorLifecycleRuntimeSymbols(),
	)
}

func ValidateActorSystemReceiveRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing actor system-message runtime object",
		RequiredActorSystemReceiveRuntimeSymbols(),
	)
}

func ValidateActorTelemetryRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing actors runtime telemetry object",
		RequiredActorTelemetryRuntimeSymbols(),
	)
}

func ValidateActorStateRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing actors runtime object",
		RequiredActorStateRuntimeSymbols(),
	)
}

func ValidateDistributedActorRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing distributed actors runtime object",
		RequiredDistributedActorRuntimeSymbols(),
	)
}

func ValidateTimeRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing time runtime object",
		RequiredTimeRuntimeSymbols(),
	)
}

func ValidateFilesystemRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing filesystem runtime object",
		RequiredFilesystemRuntimeSymbols(),
	)
}

func ValidateNetRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing networking runtime object",
		RequiredNetRuntimeSymbols(),
	)
}

func ValidateNetRuntimeObjectForSymbols(rt *tobj.Object, symbols []string) error {
	return ValidateRuntimeObjectSymbols(rt, "missing networking runtime object", symbols)
}

func ValidateSurfaceRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing surface runtime object",
		RequiredSurfaceRuntimeSymbols(),
	)
}

func ValidateTypedTaskRuntimeObject(rt *tobj.Object, maxSlots int) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing typed task runtime object",
		RequiredTypedTaskRuntimeSymbols(maxSlots),
	)
}

func ValidateTaskRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing task runtime object",
		RequiredTaskRuntimeSymbols(),
	)
}

func ValidateTaskGroupRuntimeObject(rt *tobj.Object) error {
	return ValidateRuntimeObjectSymbols(
		rt,
		"missing task group runtime object",
		RequiredTaskGroupRuntimeSymbols(),
	)
}
