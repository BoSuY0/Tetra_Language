package buildruntime_test

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/buildruntime"
)

func TestActorSystemReceiveSelectsBuiltinRuntime(t *testing.T) {
	usage := buildruntime.UsageProfile{ActorSystemReceiveUsed: true}

	got, err := buildruntime.SelectRuntimeModeForNativeTarget(
		"linux-x64",
		buildapi.RuntimeAuto,
		usage,
	)
	if err != nil {
		t.Fatalf("SelectRuntimeModeForNativeTarget(auto): %v", err)
	}
	if got != buildapi.RuntimeBuiltin {
		t.Fatalf("runtime mode = %v, want builtin", got)
	}

	_, err = buildruntime.SelectRuntimeModeForNativeTarget(
		"linux-x64",
		buildapi.RuntimeSelfHost,
		usage,
	)
	if err == nil || !strings.Contains(err.Error(), "actor system-message receive") {
		t.Fatalf("selfhost error = %v, want actor system-message receive diagnostic", err)
	}

	if buildruntime.SelfHostRuntimeSupportsNativeUsage("linux-x64", usage) {
		t.Fatalf("self-host runtime reported support for actor system-message receive")
	}
}
