package abisuite

import (
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestTargetModelChecks(t *testing.T) {
	tests := []struct {
		name   string
		target string
		check  func(ctarget.Target) error
	}{
		{name: "x86", target: "linux-x86", check: CheckX86TargetModel},
		{name: "x64", target: "linux-x64", check: CheckX64TargetModel},
		{name: "x32", target: "linux-x32", check: CheckX32TargetModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, err := ctarget.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := tt.check(tgt); err != nil {
				t.Fatalf("target model check: %v", err)
			}
		})
	}
}

func TestX86TargetModelRejectsWrongTarget(t *testing.T) {
	tgt, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	err = CheckX86TargetModel(tgt)
	if err == nil || !strings.Contains(err.Error(), "want linux-x86/linux/x86/i386-sysv") {
		t.Fatalf("CheckX86TargetModel(linux-x64) = %v", err)
	}
}
