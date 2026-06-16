package generics

import "tetra_language/compiler/internal/frontend"

func CloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func CloneFunctionTypeMap(src map[string]frontend.TypeRef) map[string]frontend.TypeRef {
	dst := make(map[string]frontend.TypeRef, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
