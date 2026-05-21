package pgrt

import (
	"errors"
	"testing"
)

func TestSCRAMSHA256RFC7677Vector(t *testing.T) {
	client, err := newSCRAMSHA256Client("user", "pencil", "rOprNGfwEbeRWgbNEkqO")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if got, want := client.ClientFirstMessage(), "n,,n=user,r=rOprNGfwEbeRWgbNEkqO"; got != want {
		t.Fatalf("client first = %q, want %q", got, want)
	}

	serverFirst := "r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"
	gotFinal, err := client.ClientFinalMessage(serverFirst)
	if err != nil {
		t.Fatalf("ClientFinalMessage: %v", err)
	}
	wantFinal := "c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,p=dHzbZapWIk4jUhN+Ute9ytag9zjfMHgsqmmiz7AndVQ="
	if gotFinal != wantFinal {
		t.Fatalf("client final = %q, want %q", gotFinal, wantFinal)
	}
	if err := client.VerifyServerFinal("v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4="); err != nil {
		t.Fatalf("VerifyServerFinal: %v", err)
	}
}

func TestSCRAMSHA256RejectsBadNonceAndSignature(t *testing.T) {
	client, err := newSCRAMSHA256Client("user", "pencil", "clientnonce")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if _, err := client.ClientFinalMessage("r=server-only,s=c2FsdA==,i=4096"); !errors.Is(err, ErrSCRAMAuthentication) {
		t.Fatalf("nonce mismatch error = %v, want ErrSCRAMAuthentication", err)
	}

	client, err = newSCRAMSHA256Client("user", "pencil", "clientnonce")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if _, err := client.ClientFinalMessage("r=clientnonce-server,s=c2FsdA==,i=4096"); err != nil {
		t.Fatalf("ClientFinalMessage: %v", err)
	}
	if err := client.VerifyServerFinal("v=YmFk"); !errors.Is(err, ErrSCRAMAuthentication) {
		t.Fatalf("bad signature error = %v, want ErrSCRAMAuthentication", err)
	}
}
