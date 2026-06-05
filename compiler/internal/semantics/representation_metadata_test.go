package semantics

import "testing"

func TestRepresentationMetadataRegistryCoversMemoryIdealV0Names(t *testing.T) {
	want := map[string]bool{
		"ptr":           true,
		"len":           true,
		"owner_id":      true,
		"region_id":     true,
		"provenance_id": true,
		"borrow_source": true,
		"storage_class": true,
		"unsafe_class":  true,
	}
	for _, field := range representationMetadataRegistry {
		if !want[field.Name] {
			t.Fatalf("unexpected representation metadata field %q", field.Name)
		}
		delete(want, field.Name)
		if field.AssignableInSafeCode {
			t.Fatalf("representation metadata field %q is assignable in safe code", field.Name)
		}
		if field.SourceFactKind != representationMetadataSourceFactKind {
			t.Fatalf("representation metadata field %q source fact kind = %q", field.Name, field.SourceFactKind)
		}
	}
	for name := range want {
		t.Fatalf("representation metadata registry missing %q", name)
	}
}

func TestRepresentationMetadataRegistryReservesMemoryIdealV0Names(t *testing.T) {
	for _, name := range []string{
		"ptr",
		"len",
		"owner_id",
		"region_id",
		"provenance_id",
		"borrow_source",
		"storage_class",
		"unsafe_class",
	} {
		if !isReservedRepresentationMetadataField(name) {
			t.Fatalf("%q is not reserved representation metadata", name)
		}
	}
}

func TestTypeModelDoesNotExposeSliceMetadataAsWritableField(t *testing.T) {
	types := baseTypes()
	info, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		t.Fatalf("ensure []u8: %v", err)
	}
	for _, name := range []string{"ptr", "len"} {
		field, ok := info.FieldMap[name]
		if !ok {
			t.Fatalf("[]u8 metadata field %q missing from internal layout", name)
		}
		if field.UserAssignable {
			t.Fatalf("[]u8 metadata field %q is user-assignable", name)
		}
	}
}

func TestTypeModelDoesNotExposeStringMetadataAsWritableField(t *testing.T) {
	types := baseTypes()
	info := types["str"]
	if info == nil {
		t.Fatal("str type missing")
	}
	for _, name := range []string{"ptr", "len"} {
		field, ok := info.FieldMap[name]
		if !ok {
			t.Fatalf("str metadata field %q missing from internal layout", name)
		}
		if field.UserAssignable {
			t.Fatalf("str metadata field %q is user-assignable", name)
		}
	}
}
