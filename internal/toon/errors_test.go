package toon

import "testing"

func assertTOONErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected TOON error %s, got nil", want)
	}
	toonErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error %s, got %T: %v", want, err, err)
	}
	if toonErr.Code != want {
		t.Fatalf("error code = %s, want %s (%v)", toonErr.Code, want, err)
	}
}
