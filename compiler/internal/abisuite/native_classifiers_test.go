package abisuite

import (
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestNativeClassifierChecks(t *testing.T) {
	tests := []struct {
		name   string
		target string
		check  func(ctarget.Target) error
	}{
		{name: "x86 classifier", target: "linux-x86", check: CheckX86I386Classifier},
		{name: "x86 varargs and sret", target: "linux-x86", check: CheckX86VarargsAndSRet},
		{name: "x64 sysv classifier", target: "linux-x64", check: CheckX64Classifier},
		{name: "x64 sysv varargs and aggregates", target: "linux-x64", check: CheckX64VarargsAndAggregates},
		{name: "x64 win64 classifier", target: "windows-x64", check: CheckX64Classifier},
		{name: "x64 win64 varargs and aggregates", target: "windows-x64", check: CheckX64VarargsAndAggregates},
		{name: "x32 classifier", target: "linux-x32", check: CheckX32SysVClassifier},
		{name: "x32 varargs and aggregates", target: "linux-x32", check: CheckX32SysVVarargsAndAggregates},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, err := ctarget.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := tt.check(tgt); err != nil {
				t.Fatalf("classifier check: %v", err)
			}
		})
	}
}
