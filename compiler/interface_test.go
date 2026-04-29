package compiler

import (
	"strings"
	"testing"
)

func TestGenerateInterfaceFromSourceWritesT4IStubs(t *testing.T) {
	src := []byte(`module math.core

struct Point:
    x: Int
    y: Int

func add(a: Int, b: Int) -> Int:
    return a + b

func enabled() -> Bool:
    return true
`)
	out, err := GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"module math.core",
		"struct Point:",
		"func add(a: i32, b: i32) -> i32:",
		"    return 0",
		"func enabled() -> bool:",
		"    return false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateInterfaceFromSourceFiltersPrivateSurfaceAndHashesPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import hidden.impl as impl
pub import public.types.{Vec}

pub struct Point:
    x: Int
    y: Int

struct Secret:
    value: Int

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 99
`)
	out, err := GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"// t4i-hash: sha256:",
		"pub import public.types.{Vec}",
		"pub struct Point:",
		"pub func add(a: i32, b: i32) -> i32:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	for _, leak := range []string{"hidden.impl", "struct Secret", "func hidden"} {
		if strings.Contains(text, leak) {
			t.Fatalf("interface leaked %q:\n%s", leak, text)
		}
	}

	out2, err := GenerateInterfaceFromSource([]byte(strings.Replace(string(src), "return 99", "return 100", 1)), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource second: %v", err)
	}
	if string(out2) != text {
		t.Fatalf("private body-only change should not change interface hash\nbefore:\n%s\nafter:\n%s", text, out2)
	}
}

func TestInterfaceFingerprintFromSourceIsPublicAPIStable(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 1
`)
	hash1, err := InterfaceFingerprintFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	privateBodyChanged := []byte(strings.Replace(string(src), "return 1", "return 2", 1))
	hash2, err := InterfaceFingerprintFromSource(privateBodyChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource private change: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("private implementation change changed public API hash: %s vs %s", hash1, hash2)
	}
	publicSigChanged := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	hash3, err := InterfaceFingerprintFromSource(publicSigChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource public change: %v", err)
	}
	if hash1 == hash3 {
		t.Fatalf("public signature change did not change API hash: %s", hash1)
	}
}

func TestValidateInterfaceAgainstSourceReportsPublicAPIMismatch(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	iface, err := GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	changedSource := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	err = ValidateInterfaceAgainstSource(changedSource, iface, "math/core.t4")
	if err == nil {
		t.Fatalf("expected public API mismatch")
	}
	if !strings.Contains(err.Error(), "public API mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredByPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import math.types as mt
import hidden.impl as hidden

pub func norm(v: mt.Vec) -> Int:
    return v.x

func private_helper(v: hidden.Secret) -> Int:
    return 0
`)
	iface, err := GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	if !strings.Contains(text, "import math.types as mt") {
		t.Fatalf("interface omitted public-signature import:\n%s", text)
	}
	if strings.Contains(text, "hidden.impl") {
		t.Fatalf("interface leaked private-only import:\n%s", text)
	}
}

func TestInterfaceFingerprintFromSourceTracksHashOnlyPublicSurface(t *testing.T) {
	src := []byte(`module app.config

pub const build: Int = 1
`)
	hash1, err := InterfaceFingerprintFromSource(src, "app/config.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	hash2, err := InterfaceFingerprintFromSource([]byte(strings.Replace(string(src), "build: Int", "build: Bool", 1)), "app/config.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource changed: %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("public hash-only global surface change did not change API hash: %s", hash1)
	}
}
