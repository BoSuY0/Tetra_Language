package toon

import "testing"

func TestDecodeRejectsDuplicateKeys(t *testing.T) {
	_, err := ConvertTOONToJSON([]byte("name: Ada\nname: Grace"), Options{Strict: true})
	assertTOONErrorCode(t, err, CodeDuplicateKey)
}

func TestDecodeRejectsTabularRowCountMismatch(t *testing.T) {
	input := []byte("users[2]{id,name}:\n  1,Ada")
	_, err := ConvertTOONToJSON(input, Options{Strict: true})
	assertTOONErrorCode(t, err, CodeRowCountMismatch)
}

func TestDecodeRejectsTabularColumnCountMismatch(t *testing.T) {
	input := []byte("users[1]{id,name}:\n  1")
	_, err := ConvertTOONToJSON(input, Options{Strict: true})
	assertTOONErrorCode(t, err, CodeColumnCountMismatch)
}

func TestDecodeRejectsInvalidUTF8(t *testing.T) {
	_, err := ConvertTOONToJSON([]byte{0xff, 0xfe}, Options{Strict: true})
	assertTOONErrorCode(t, err, CodeInvalidUTF8)
}

func TestDecodeRejectsMultipleTopLevelScalars(t *testing.T) {
	_, err := ConvertTOONToJSON([]byte("hello\nworld"), Options{Strict: true})
	assertTOONErrorCode(t, err, CodeMultipleTopLevel)
}
