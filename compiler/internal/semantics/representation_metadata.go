package semantics

type RepresentationMetadataField struct {
	Name                 string
	AppliesToTypes       []TypeKind
	ReadableInSafeCode   bool
	AssignableInSafeCode bool
	SourceFactKind       string
}

const representationMetadataSourceFactKind = "safe_representation_metadata:not_user_assignable"

var representationMetadataRegistry = []RepresentationMetadataField{
	{Name: "ptr", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, ReadableInSafeCode: true, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "len", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, ReadableInSafeCode: true, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "owner_id", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "region_id", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "provenance_id", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "borrow_source", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "storage_class", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
	{Name: "unsafe_class", AppliesToTypes: []TypeKind{TypeSlice, TypeArray, TypeStr}, SourceFactKind: representationMetadataSourceFactKind},
}

func representationMetadataByName(name string) (RepresentationMetadataField, bool) {
	for _, field := range representationMetadataRegistry {
		if field.Name == name {
			return field, true
		}
	}
	return RepresentationMetadataField{}, false
}

func isReservedRepresentationMetadataField(field string) bool {
	_, ok := representationMetadataByName(field)
	return ok
}
