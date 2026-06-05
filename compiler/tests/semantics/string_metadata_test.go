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

func TestStringMetadataAssignmentRejectsNestedPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    text: String

func main() -> Int:
    var box: Box = Box(text: "*")
    box.text.ptr = 0
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsGenericNestedLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box<T>:
    value: T

func main() -> Int:
    var box: Box<String> = Box<String>{value: "*"}
    box.value.len = 2
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsOptionalPayloadPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
    var maybe: String? = "*"
    if let some(text) = maybe:
        text.ptr = 0
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsEnumPayloadLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum TextMsg:
    case text(String)

func main() -> Int:
    var msg: TextMsg = TextMsg.text("*")
    match msg:
        case TextMsg.text(text):
            text.len = 9
    return 0
`, "cannot assign to string internals ('ptr'/'len')")
}

func TestStringMetadataAssignmentRejectsInoutParameterLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func poke(text: inout String) -> Int:
    text.len = 9
    return 0

func main() -> Int:
    var text: String = "*"
    return poke(text)
`, "cannot assign to string internals ('ptr'/'len')")
}
