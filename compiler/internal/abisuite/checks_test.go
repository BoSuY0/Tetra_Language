package abisuite

import (
	"errors"
	"strings"
	"testing"
)

func TestRunChecksRecordsErrorsAndContinues(t *testing.T) {
	var order []string
	checks := RunChecks([]Case{
		{Name: "first", Run: func() error {
			order = append(order, "first")
			return nil
		}},
		{Name: "second", Run: func() error {
			order = append(order, "second")
			return errors.New("boom")
		}},
	})

	if len(checks) != 2 {
		t.Fatalf("checks len = %d, want 2", len(checks))
	}
	if checks[0] != (Check{Name: "first"}) {
		t.Fatalf("first check = %#v, want success", checks[0])
	}
	if checks[1].Name != "second" || checks[1].Error != "boom" {
		t.Fatalf("second check = %#v, want recorded error", checks[1])
	}
	if strings.Join(order, ",") != "first,second" {
		t.Fatalf("run order = %q, want first,second", strings.Join(order, ","))
	}
}

func TestUnsupportedTargetError(t *testing.T) {
	err := UnsupportedTargetError("plan9-x64")
	if err == nil || !strings.Contains(err.Error(), "ABI suite for target plan9-x64 is not implemented") {
		t.Fatalf("UnsupportedTargetError = %v", err)
	}
}
