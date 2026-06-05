package semantics

import "tetra_language/compiler/internal/frontend"

type callableEscapeBoundary string

const (
	callableBoundaryLocal       callableEscapeBoundary = "local"
	callableBoundaryReturn      callableEscapeBoundary = "return"
	callableBoundaryGlobal      callableEscapeBoundary = "global"
	callableBoundaryStructField callableEscapeBoundary = "struct-field"
	callableBoundaryEnumPayload callableEscapeBoundary = "enum-payload"
	callableBoundaryCallback    callableEscapeBoundary = "callback"
	callableBoundaryThread      callableEscapeBoundary = "thread"
)

func classifyCallableEscape(
	boundary callableEscapeBoundary,
	captures []frontend.ClosureCapture,
	types map[string]*TypeInfo,
) (CallableEscapeKind, bool, error) {
	slots, err := functionCaptureSlotCount(captures, types)
	if err != nil {
		return "", false, err
	}
	if capture, surfaceType, ok := surfaceEphemeralCallableCapture(captures, types); ok {
		return "", false, lifetimeDiagnosticf(capture.At, "surface value '%s' cannot escape via function capture; keep Surface Frame/Event/DrawContext values local to the active Surface turn", surfaceType)
	}
	if slots <= FnPtrEnvSlotCount && boundary != callableBoundaryThread {
		return CallableEscapeLocalSnapshot, false, nil
	}

	escapeKind := CallableEscapeHeap
	if boundary == callableBoundaryGlobal {
		escapeKind = CallableEscapeGlobal
	}
	if boundary == callableBoundaryThread {
		escapeKind = CallableEscapeThread
	}
	for _, capture := range captures {
		if capture.Mutable {
			return "", false, unsupportedCallableMutableCaptureEscapeError(capture.At, escapeKind, capture.Name)
		}
		if _, err := ensureTypeInfo(capture.Type.Name, types); err != nil {
			return "", false, err
		}
		if !isClosureCaptureType(capture.Type.Name, types) {
			return "", false, unsupportedCallableResourceCaptureEscapeError(capture.At, capture.Name, capture.Type.Name)
		}
	}
	return escapeKind, true, nil
}

func surfaceEphemeralCallableCapture(captures []frontend.ClosureCapture, types map[string]*TypeInfo) (frontend.ClosureCapture, string, bool) {
	for _, capture := range captures {
		if surfaceType, ok := surfaceEphemeralValueType(capture.Type.Name, types); ok {
			return capture, surfaceType, true
		}
	}
	return frontend.ClosureCapture{}, "", false
}
