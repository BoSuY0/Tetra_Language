package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestStringMetadataAssignmentRejectsLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    var text: String = "*"
    text.len = 2
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    var tiny: String = "*"
    var wide: String = "AB"
    wide.ptr = tiny.ptr
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsNestedLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    text: String

func main() -> Int:
    var box: Box = Box(text: "*")
    box.text.len = 2
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}
